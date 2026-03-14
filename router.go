package main

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// DispatchError is returned by Dispatch when a request is rejected.
// Message is the user-facing string that should be sent back to Slack.
type DispatchError struct {
	Reason  string // "acl" | "global" | "route"
	Message string // ready-to-send Slack message (from config, with defaults)
}

func (e *DispatchError) Error() string { return e.Reason }

// Router dispatches incoming slash command events to worker processes,
// enforcing global and per-route concurrency limits.
type Router struct {
	cfg       *Config
	routes    map[string]*RouteConfig // command → config
	globalSem chan struct{}            // global concurrency gate
	routeSems map[string]chan struct{} // per-command concurrency gates

	// mu guards shuttingDown and must be held when calling wg.Add.
	// This prevents wg.Add from racing with wg.Wait during shutdown.
	mu           sync.Mutex
	shuttingDown bool
	wg           sync.WaitGroup

	// notifyWg tracks fire-and-forget notification goroutines (e.g. ephemeral
	// Slack messages). Unlike wg, it is not subject to shuttingDown — we want
	// in-flight notifications to complete even during shutdown.
	notifyWg sync.WaitGroup
}

// NewRouter creates a Router from the given configuration.
func NewRouter(cfg *Config) *Router {
	routes := make(map[string]*RouteConfig, len(cfg.Routes))
	routeSems := make(map[string]chan struct{}, len(cfg.Routes))

	for i := range cfg.Routes {
		r := &cfg.Routes[i]
		routes[r.Command] = r
		if r.MaxConcurrency > 0 {
			routeSems[r.Command] = make(chan struct{}, r.MaxConcurrency)
		}
	}

	return &Router{
		cfg:       cfg,
		routes:    routes,
		globalSem: make(chan struct{}, cfg.Global.MaxConcurrentWorkers),
		routeSems: routeSems,
	}
}

// Dispatch routes event to the matching worker script.
//
// If concurrency limits are hit the request is dropped and the caller
// receives a non-nil error; it should relay that to the user via
// response_url. When the limits are satisfied, Dispatch returns nil
// immediately and the worker runs asynchronously.
func (r *Router) Dispatch(ctx context.Context, event SlashEvent) error {
	route, ok := r.routes[event.Command]
	if !ok {
		// No route means a Slack command was registered without a matching
		// config entry; log and ignore silently (no user message).
		slog.Warn("no route configured", "command", event.Command)
		return nil
	}

	// ACL check — evaluated before touching any semaphore.
	if !route.ACL.isEmpty() {
		if err := route.ACL.Check(event.UserID, event.ChannelID); err != nil {
			var denied *aclDenied
			if errors.As(err, &denied) {
				slog.Warn("request denied by ACL",
					"command", event.Command,
					"user", event.UserID,
					"channel", event.ChannelID,
					"reason", denied.reason,
				)
			}
			return &DispatchError{Reason: "acl", Message: route.DenyMessage}
		}
	}

	// Acquire global semaphore (non-blocking).
	select {
	case r.globalSem <- struct{}{}:
	default:
		slog.Warn("global concurrency limit reached, dropping request",
			"command", event.Command, "user", event.UserID,
			"limit", r.cfg.Global.MaxConcurrentWorkers)
		return &DispatchError{Reason: "global", Message: r.cfg.Global.Messages.ServerBusy}
	}

	// Acquire per-route semaphore (non-blocking).
	if sem, hasSem := r.routeSems[event.Command]; hasSem {
		select {
		case sem <- struct{}{}:
		default:
			<-r.globalSem // release global slot we just took
			slog.Warn("route concurrency limit reached, dropping request",
				"command", event.Command, "user", event.UserID,
				"limit", route.MaxConcurrency)
			return &DispatchError{Reason: "route", Message: route.BusyMessage}
		}
	}

	// Guard wg.Add with the shutdown mutex so it cannot race with Wait.
	// If shutdown has already begun, silently drop the request — the user
	// will not receive a response, but the process is exiting anyway.
	r.mu.Lock()
	if r.shuttingDown {
		r.mu.Unlock()
		<-r.globalSem // release the slot we just acquired above
		if sem, hasSem := r.routeSems[event.Command]; hasSem {
			<-sem
		}
		slog.Info("router: shutdown in progress, dropping request", "command", event.Command)
		return nil
	}
	r.wg.Add(1)
	r.mu.Unlock()

	go func() {
		defer r.wg.Done()
		defer func() { <-r.globalSem }()
		if sem, hasSem := r.routeSems[event.Command]; hasSem {
			defer func() { <-sem }()
		}
		runWorker(ctx, route.Script, route.Timeout, event)
	}()

	return nil
}

// GoNotify runs fn in a tracked goroutine. Use this for fire-and-forget
// notifications (e.g. notifyEphemeral) so they are guaranteed to complete
// before the process exits, even when called close to shutdown time.
func (r *Router) GoNotify(fn func()) {
	r.notifyWg.Add(1)
	go func() {
		defer r.notifyWg.Done()
		fn()
	}()
}

// Wait signals that no new workers should be accepted, then blocks until all
// running workers and in-flight notifications have exited. It must be called
// at most once.
func (r *Router) Wait() {
	r.mu.Lock()
	r.shuttingDown = true
	r.mu.Unlock()

	r.wg.Wait()
	r.notifyWg.Wait()
}
