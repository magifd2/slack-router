package main

import (
	"fmt"
	"slices"
)

// ACL holds per-route access control lists.
//
// Evaluation order (highest precedence first):
//  1. deny_users    — listed users are always rejected
//  2. deny_channels — listed channels are always rejected
//  3. allow_users   — if non-empty, unlisted users are rejected
//  4. allow_channels — if non-empty, unlisted channels are rejected
//
// An empty allow list means "allow all"; a non-empty allow list is a whitelist.
// Deny lists take precedence over allow lists.
type ACL struct {
	AllowChannels []string `yaml:"allow_channels"`
	AllowUsers    []string `yaml:"allow_users"`
	DenyChannels  []string `yaml:"deny_channels"`
	DenyUsers     []string `yaml:"deny_users"`
}

type aclDenied struct{ reason string }

func (e *aclDenied) Error() string { return e.reason }

// Check returns an *aclDenied error if the request should be rejected,
// or nil if it is permitted.
func (a *ACL) Check(userID, channelID string) error {
	if slices.Contains(a.DenyUsers, userID) {
		return &aclDenied{reason: fmt.Sprintf("user %q is in the deny list", userID)}
	}
	if slices.Contains(a.DenyChannels, channelID) {
		return &aclDenied{reason: fmt.Sprintf("channel %q is in the deny list", channelID)}
	}
	if len(a.AllowUsers) > 0 && !slices.Contains(a.AllowUsers, userID) {
		return &aclDenied{reason: fmt.Sprintf("user %q is not in the allow list", userID)}
	}
	if len(a.AllowChannels) > 0 && !slices.Contains(a.AllowChannels, channelID) {
		return &aclDenied{reason: fmt.Sprintf("channel %q is not in the allow list", channelID)}
	}
	return nil
}

// isEmpty reports whether no ACL rules are configured (i.e. everything is allowed).
func (a *ACL) isEmpty() bool {
	return len(a.AllowChannels) == 0 &&
		len(a.AllowUsers) == 0 &&
		len(a.DenyChannels) == 0 &&
		len(a.DenyUsers) == 0
}

