// NOTE: Do not import this package in non-test code!
package apitest

import (
	"io"
	"log/slog"
	"testing"
	"testing/fstest"

	"github.com/thepeacockproject/Octopussy/internal/api"
	"github.com/thepeacockproject/Octopussy/internal/content"
	"github.com/thepeacockproject/Octopussy/internal/controller"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
	"github.com/thepeacockproject/Octopussy/internal/store"
)

// NewTestController builds an api backed by an in-memory store and a
// fresh (ephemeral-key) JWT manager. It returns the api and a helper
// that mints a bearer token for the given profile id.
func NewTestController(t *testing.T) (*controller.Controller, func(userId string) string) {
	t.Helper()

	st, err := store.Open(store.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	mgr, err := jwt.NewManager(jwt.Config{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}

	c := controller.New(slog.New(slog.NewTextHandler(io.Discard, nil)), mgr, st, testContent(t))
	api.RegisterRoutes(c)

	token := func(userId string) string {
		tok, err := mgr.Sign(jwt.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: userId}})
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return tok
	}

	return c, token
}

// testContent builds a content store from a tiny in-memory data layout, so
// api tests don't depend on the real game data files.
func testContent(t *testing.T) *content.Store {
	t.Helper()
	fsys := fstest.MapFS{
		"menu/mission_lists.json": {Data: []byte(
			`[{"Name":"KNT_Missions","Missions":["aaaaaaaa-0000-4000-8000-000000000001"]}]`)},
		"menu/player_levels.json": {Data: []byte(`{"XpThresholds":[3950,7900]}`)},
		"missions/aaaaaaaa-0000-4000-8000-000000000001.json": {Data: []byte(
			`{"id":"aaaaaaaa-0000-4000-8000-000000000001","Ui":{"Title":"Test Mission"},"Group":null}`)},
		"privacypolicy_pc_en.json": {Data: []byte(`{"view":"menu3.modal.ModalDialogGeneric","type":"gdprconsent"}`)},
		"challenges.json":          {Data: []byte(`[]`)},
		"unlockables.json":         {Data: []byte(`[]`)},
	}
	st, err := content.Load(fsys)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	return st
}
