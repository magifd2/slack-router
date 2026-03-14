package main

import "testing"

func TestACLCheck(t *testing.T) {
	tests := []struct {
		name      string
		acl       ACL
		userID    string
		channelID string
		wantErr   bool
	}{
		// ── 制限なし ────────────────────────────────────────────────
		{
			name:      "empty ACL allows everything",
			acl:       ACL{},
			userID:    "U001",
			channelID: "C001",
			wantErr:   false,
		},

		// ── deny_users ──────────────────────────────────────────────
		{
			name:      "deny_users blocks listed user",
			acl:       ACL{DenyUsers: []string{"U001"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},
		{
			name:      "deny_users allows unlisted user",
			acl:       ACL{DenyUsers: []string{"U002"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   false,
		},

		// ── deny_channels ───────────────────────────────────────────
		{
			name:      "deny_channels blocks listed channel",
			acl:       ACL{DenyChannels: []string{"C001"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},
		{
			name:      "deny_channels allows unlisted channel",
			acl:       ACL{DenyChannels: []string{"C002"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   false,
		},

		// ── allow_users ─────────────────────────────────────────────
		{
			name:      "allow_users permits listed user",
			acl:       ACL{AllowUsers: []string{"U001"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   false,
		},
		{
			name:      "allow_users blocks unlisted user",
			acl:       ACL{AllowUsers: []string{"U002"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},

		// ── allow_channels ──────────────────────────────────────────
		{
			name:      "allow_channels permits listed channel",
			acl:       ACL{AllowChannels: []string{"C001"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   false,
		},
		{
			name:      "allow_channels blocks unlisted channel",
			acl:       ACL{AllowChannels: []string{"C002"}},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},

		// ── deny が allow より優先される ────────────────────────────
		{
			name: "deny_users takes precedence over allow_users",
			acl: ACL{
				AllowUsers: []string{"U001"},
				DenyUsers:  []string{"U001"},
			},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},
		{
			name: "deny_channels takes precedence over allow_channels",
			acl: ACL{
				AllowChannels: []string{"C001"},
				DenyChannels:  []string{"C001"},
			},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},

		// ── 複合パターン ────────────────────────────────────────────
		{
			name: "user allowed but channel denied",
			acl: ACL{
				AllowUsers:   []string{"U001"},
				DenyChannels: []string{"C001"},
			},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},
		{
			name: "user and channel both in allow lists",
			acl: ACL{
				AllowUsers:    []string{"U001"},
				AllowChannels: []string{"C001"},
			},
			userID:    "U001",
			channelID: "C001",
			wantErr:   false,
		},
		{
			name: "user in allow list but channel not",
			acl: ACL{
				AllowUsers:    []string{"U001"},
				AllowChannels: []string{"C002"},
			},
			userID:    "U001",
			channelID: "C001",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.acl.Check(tt.userID, tt.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestACLIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		acl  ACL
		want bool
	}{
		{"empty ACL", ACL{}, true},
		{"has allow_users", ACL{AllowUsers: []string{"U001"}}, false},
		{"has allow_channels", ACL{AllowChannels: []string{"C001"}}, false},
		{"has deny_users", ACL{DenyUsers: []string{"U001"}}, false},
		{"has deny_channels", ACL{DenyChannels: []string{"C001"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.acl.isEmpty(); got != tt.want {
				t.Errorf("isEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
