package content

import "testing"

func TestLevelForXP(t *testing.T) {
	// Thresholds [3950, 7900]: level 1 below 3950, level 2 at/above it, level 3
	// at/above 7900.
	s, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cases := []struct {
		xp   int64
		want int
	}{
		{0, 1},
		{3949, 1},
		{3950, 2},
		{5500, 2},
		{7899, 2},
		{7900, 3},
		{1_000_000, 3},
	}
	for _, tc := range cases {
		if got := s.LevelForXP(tc.xp); got != tc.want {
			t.Errorf("LevelForXP(%d) = %d, want %d", tc.xp, got, tc.want)
		}
	}
}
