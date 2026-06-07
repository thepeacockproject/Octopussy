package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

// seedUser inserts a user row directly so sessions have a valid users.id to
// reference, with the intel-currency and rank values a session should snapshot.
func seedUser(t *testing.T, st *Store, id string, intel, rank int) User {
	t.Helper()
	u := User{ID: id, Gamertag: "gamer", IntelCurrency: intel, Rank: rank}
	if err := st.db.WithContext(context.Background()).Create(&u).Error; err != nil {
		t.Fatalf("seed user %q: %v", id, err)
	}
	return u
}

func TestCreateSessionSnapshotsUser(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	user := seedUser(t, st, "profile-1", 250, 5)

	sess, err := st.CreateSession(ctx, user, "contract-9")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if sess.SessionID == "" {
		t.Error("SessionID is empty, want a generated UUID")
	}
	if sess.UserID != "profile-1" {
		t.Errorf("UserID = %q, want %q", sess.UserID, "profile-1")
	}
	if sess.ContractID != "contract-9" {
		t.Errorf("ContractID = %q, want %q", sess.ContractID, "contract-9")
	}
	if sess.IntelCurrencyWas != 250 {
		t.Errorf("IntelCurrencyWas = %d, want 250", sess.IntelCurrencyWas)
	}
	if sess.RankWas != 5 {
		t.Errorf("RankWas = %d, want 5", sess.RankWas)
	}

	// The row must actually be persisted.
	var count int64
	st.db.WithContext(ctx).Model(&UserSession{}).Where("session_id = ?", sess.SessionID).Count(&count)
	if count != 1 {
		t.Errorf("persisted session count = %d, want 1", count)
	}
}

func TestCreateSessionGeneratesUniqueIDs(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	user := seedUser(t, st, "profile-1", 0, 1)

	first, err := st.CreateSession(ctx, user, "c1")
	if err != nil {
		t.Fatalf("CreateSession #1: %v", err)
	}
	second, err := st.CreateSession(ctx, user, "c2")
	if err != nil {
		t.Fatalf("CreateSession #2: %v", err)
	}
	if first.SessionID == second.SessionID {
		t.Errorf("two sessions share id %q, want distinct ids", first.SessionID)
	}
}

func TestGetSessionByID(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	user := seedUser(t, st, "profile-1", 250, 5)

	created, err := st.CreateSession(ctx, user, "contract-9")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := st.GetSessionByID(ctx, created.SessionID)
	if err != nil {
		t.Fatalf("GetSessionByID: %v", err)
	}
	if got.SessionID != created.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, created.SessionID)
	}
	if got.UserID != "profile-1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "profile-1")
	}
	if got.ContractID != "contract-9" {
		t.Errorf("ContractID = %q, want %q", got.ContractID, "contract-9")
	}
	if got.IntelCurrencyWas != 250 || got.RankWas != 5 {
		t.Errorf("snapshot = (%d, %d), want (250, 5)", got.IntelCurrencyWas, got.RankWas)
	}
}

func TestGetSessionByIDUnknownReturnsNotFound(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// GetSessionByID uses First, so an unknown id surfaces as ErrRecordNotFound.
	_, err := st.GetSessionByID(ctx, "does-not-exist")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetSessionByID(unknown) error = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestGetCurrentSessionByUser(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	user := seedUser(t, st, "profile-1", 100, 2)

	older, err := st.CreateSession(ctx, user, "first")
	if err != nil {
		t.Fatalf("CreateSession #1: %v", err)
	}
	latest, err := st.CreateSession(ctx, user, "second")
	if err != nil {
		t.Fatalf("CreateSession #2: %v", err)
	}

	// Both sessions are created in the same instant, so pin distinct created_at
	// values to make "most recent" unambiguous and actually exercise the ordering.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := st.db.WithContext(ctx).Model(&UserSession{}).
		Where("session_id = ?", older.SessionID).Update("created_at", base).Error; err != nil {
		t.Fatalf("set older created_at: %v", err)
	}
	if err := st.db.WithContext(ctx).Model(&UserSession{}).
		Where("session_id = ?", latest.SessionID).Update("created_at", base.Add(time.Hour)).Error; err != nil {
		t.Fatalf("set latest created_at: %v", err)
	}

	got, err := st.GetCurrentSessionByUser(ctx, "profile-1")
	if err != nil {
		t.Fatalf("GetCurrentSessionByUser: %v", err)
	}
	if got.UserID != "profile-1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "profile-1")
	}
	if got.SessionID != latest.SessionID {
		t.Errorf("SessionID = %q, want the most recent session %q", got.SessionID, latest.SessionID)
	}
}
