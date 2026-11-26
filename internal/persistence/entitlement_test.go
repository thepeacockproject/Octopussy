package persistence

import (
	"context"
	"strings"
	"testing"
)

func TestEntitlementsPerIdentity(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// Two accounts linked to one profile each keep their own set.
	for _, steamId := range []string{"111", "222"} {
		if err := st.EnsureUser(ctx, "profile-1", "gamer", "steam", steamId); err != nil {
			t.Fatalf("EnsureUser %s: %v", steamId, err)
		}
	}

	sets := []struct {
		steamId string
		set     []string
	}{
		{"111", []string{"b", "a"}},
		{"222", []string{"x"}},
		{"111", []string{"c"}},
	}
	for _, s := range sets {
		if err := st.SetEntitlements(ctx, "steam", s.steamId, s.set); err != nil {
			t.Fatalf("SetEntitlements %s: %v", s.steamId, err)
		}
	}

	for steamId, want := range map[string]string{"111": "c", "222": "x"} {
		got, err := st.GetEntitlements(ctx, "steam", steamId)
		if err != nil {
			t.Fatalf("GetEntitlements %s: %v", steamId, err)
		}
		if strings.Join(got, ",") != want {
			t.Errorf("%s entitlements = %v, want [%s]", steamId, got, want)
		}
	}
}
