package persistence

import (
	"context"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestEnsureUserCreatesUserAndIdentity(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	if err := st.EnsureUser(ctx, "profile-1", "gamer", "steam", "steam-123"); err != nil {
		t.Fatalf("EnsureUser: %v", err)
	}

	var user User
	if err := st.db.WithContext(ctx).First(&user, "id = ?", "profile-1").Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Gamertag != "gamer" {
		t.Errorf("gamertag = %q, want %q", user.Gamertag, "gamer")
	}
	if user.XPEarned != 0 {
		t.Errorf("xp_earned = %d, want 0 by default", user.XPEarned)
	}

	var identity UserIdentity
	if err := st.db.WithContext(ctx).First(&identity, "platform = ? AND platform_id = ?", "steam", "steam-123").Error; err != nil {
		t.Fatalf("load identity: %v", err)
	}
	if identity.UserId != "profile-1" {
		t.Errorf("identity.UserId = %q, want %q", identity.UserId, "profile-1")
	}
}

func TestEnsureUserIsIdempotent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := st.EnsureUser(ctx, "profile-1", "gamer", "steam", "steam-123"); err != nil {
			t.Fatalf("EnsureUser call %d: %v", i, err)
		}
	}

	var users, identities int64
	st.db.WithContext(ctx).Model(&User{}).Count(&users)
	st.db.WithContext(ctx).Model(&UserIdentity{}).Count(&identities)
	if users != 1 {
		t.Errorf("user count = %d, want 1", users)
	}
	if identities != 1 {
		t.Errorf("identity count = %d, want 1", identities)
	}

	// A repeat login must not overwrite an existing gamertag.
	if err := st.EnsureUser(ctx, "profile-1", "changed-name", "steam", "steam-123"); err != nil {
		t.Fatalf("EnsureUser: %v", err)
	}
	var user User
	if err := st.db.WithContext(ctx).First(&user, "id = ?", "profile-1").Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Gamertag != "gamer" {
		t.Errorf("gamertag = %q, want it left as %q", user.Gamertag, "gamer")
	}
}

func TestUserXP(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	user, err := st.GetUser(ctx, "nobody")
	if err == nil {
		t.Fatalf("found user 'nobody', expected nil")
	}

	if user.XPEarned != 0 {
		t.Errorf("user.XPEarned = %d, want 0", user.XPEarned)
	}

	if err := st.EnsureUser(ctx, "profile-1", "gamer", "steam", "steam-123"); err != nil {
		t.Fatalf("EnsureUser: %v", err)
	}

	if err := st.db.WithContext(ctx).Model(&User{}).Where("id = ?", "profile-1").Update("xp_earned", 5500).Error; err != nil {
		t.Fatalf("set xp: %v", err)
	}

	user, err = st.GetUser(ctx, "profile-1")
	if err != nil {
		t.Fatalf("UserXP: %v", err)
	}
	if user.XPEarned != 5500 {
		t.Errorf("UserXP = %d, want 5500", user.XPEarned)
	}
}

func TestEnsureUserLinksSecondIdentity(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	if err := st.EnsureUser(ctx, "profile-1", "gamer", "steam", "steam-123"); err != nil {
		t.Fatalf("EnsureUser steam: %v", err)
	}
	if err := st.EnsureUser(ctx, "profile-1", "gamer", "epic", "epic-456"); err != nil {
		t.Fatalf("EnsureUser epic: %v", err)
	}

	var users, identities int64
	st.db.WithContext(ctx).Model(&User{}).Count(&users)
	st.db.WithContext(ctx).Model(&UserIdentity{}).Count(&identities)
	if users != 1 {
		t.Errorf("user count = %d, want 1", users)
	}
	if identities != 2 {
		t.Errorf("identity count = %d, want 2", identities)
	}
}
