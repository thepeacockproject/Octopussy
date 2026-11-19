package persistence

import (
	"context"
	"testing"
)

// the game can resend CompleteActiveChallenge for a challenge that's already
// recorded. the second insert must succeed and report created=false.
func TestCreateChallengeIsIdempotent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	seedUser(t, st, "profile-1", 0, 1)

	_, created, err := st.CreateChallenge(ctx, "profile-1", "challenge-1")
	if err != nil {
		t.Fatalf("first CreateChallenge: %v", err)
	}
	if !created {
		t.Error("first CreateChallenge: created = false, want true")
	}

	_, created, err = st.CreateChallenge(ctx, "profile-1", "challenge-1")
	if err != nil {
		t.Fatalf("second CreateChallenge: %v", err)
	}
	if created {
		t.Error("second CreateChallenge: created = true, want false")
	}

	rows, err := st.GetChallenges(ctx, "profile-1")
	if err != nil {
		t.Fatalf("GetChallenges: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("got %d challenge rows, want 1", len(rows))
	}
}
