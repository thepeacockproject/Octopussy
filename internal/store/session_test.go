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
	u := User{Id: id, Gamertag: "gamer", IntelCurrency: intel, Rank: rank}
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

	if sess.SessionId == "" {
		t.Error("SessionId is empty, want a generated UUID")
	}
	if sess.UserId != "profile-1" {
		t.Errorf("UserId = %q, want %q", sess.UserId, "profile-1")
	}
	if sess.ContractId != "contract-9" {
		t.Errorf("ContractId = %q, want %q", sess.ContractId, "contract-9")
	}
	if sess.IntelCurrencyWas != 250 {
		t.Errorf("IntelCurrencyWas = %d, want 250", sess.IntelCurrencyWas)
	}
	if sess.RankWas != 5 {
		t.Errorf("RankWas = %d, want 5", sess.RankWas)
	}

	// The row must actually be persisted.
	var count int64
	st.db.WithContext(ctx).Model(&UserSession{}).Where("session_id = ?", sess.SessionId).Count(&count)
	if count != 1 {
		t.Errorf("persisted session count = %d, want 1", count)
	}
}

func TestCreateSessionGeneratesUniqueIds(t *testing.T) {
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
	if first.SessionId == second.SessionId {
		t.Errorf("two sessions share id %q, want distinct ids", first.SessionId)
	}
}

func TestGetSessionById(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	user := seedUser(t, st, "profile-1", 250, 5)

	created, err := st.CreateSession(ctx, user, "contract-9")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := st.GetSessionById(ctx, created.SessionId)
	if err != nil {
		t.Fatalf("GetSessionById: %v", err)
	}
	if got.SessionId != created.SessionId {
		t.Errorf("SessionId = %q, want %q", got.SessionId, created.SessionId)
	}
	if got.UserId != "profile-1" {
		t.Errorf("UserId = %q, want %q", got.UserId, "profile-1")
	}
	if got.ContractId != "contract-9" {
		t.Errorf("ContractId = %q, want %q", got.ContractId, "contract-9")
	}
	if got.IntelCurrencyWas != 250 || got.RankWas != 5 {
		t.Errorf("snapshot = (%d, %d), want (250, 5)", got.IntelCurrencyWas, got.RankWas)
	}
}

func TestGetSessionByIdUnknownReturnsNotFound(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// GetSessionById uses First, so an unknown id surfaces as ErrRecordNotFound.
	_, err := st.GetSessionById(ctx, "does-not-exist")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetSessionById(unknown) error = %v, want gorm.ErrRecordNotFound", err)
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
		Where("session_id = ?", older.SessionId).Update("created_at", base).Error; err != nil {
		t.Fatalf("set older created_at: %v", err)
	}
	if err := st.db.WithContext(ctx).Model(&UserSession{}).
		Where("session_id = ?", latest.SessionId).Update("created_at", base.Add(time.Hour)).Error; err != nil {
		t.Fatalf("set latest created_at: %v", err)
	}

	got, err := st.GetCurrentSessionByUser(ctx, "profile-1")
	if err != nil {
		t.Fatalf("GetCurrentSessionByUser: %v", err)
	}
	if got.UserId != "profile-1" {
		t.Errorf("UserId = %q, want %q", got.UserId, "profile-1")
	}
	if got.SessionId != latest.SessionId {
		t.Errorf("SessionId = %q, want the most recent session %q", got.SessionId, latest.SessionId)
	}
}
