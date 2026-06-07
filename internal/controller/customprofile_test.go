package controller

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/thepeacockproject/Octopussy/internal/content"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
	"github.com/thepeacockproject/Octopussy/internal/store"
)

// testContent builds a content store from a tiny in-memory data layout, so
// controller tests don't depend on the real game data files.
func testContent(t *testing.T) *content.Store {
	t.Helper()
	fsys := fstest.MapFS{
		"menu/mission_lists.json": {Data: []byte(
			`[{"Name":"KNT_Missions","Missions":["aaaaaaaa-0000-4000-8000-000000000001"]}]`)},
		"menu/player_levels.json": {Data: []byte(`{"XpThresholds":[3950,7900]}`)},
		"missions/aaaaaaaa-0000-4000-8000-000000000001.json": {Data: []byte(
			`{"id":"aaaaaaaa-0000-4000-8000-000000000001","Ui":{"Title":"Test Mission"},"Group":null}`)},
		"privacypolicy_pc_en.json": {Data: []byte(`{"view":"menu3.modal.ModalDialogGeneric","type":"gdprconsent"}`)},
	}
	st, err := content.Load(fsys)
	if err != nil {
		t.Fatalf("load content: %v", err)
	}
	return st
}

// newTestController builds a controller backed by an in-memory store and a
// fresh (ephemeral-key) JWT manager. It returns the controller and a helper
// that mints a bearer token for the given profile id.
func newTestController(t *testing.T) (*Controller, func(profileID string) string) {
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

	c := New(slog.New(slog.NewTextHandler(io.Discard, nil)), mgr, st, testContent(t))

	token := func(profileID string) string {
		tok, err := mgr.Sign(jwt.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: profileID}})
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return tok
	}
	return c, token
}

func TestCustomProfileReadAllRequiresToken(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/CustomProfileDataService/ReadAll", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestCustomProfileWritePersists(t *testing.T) {
	c, token := newTestController(t)
	const profileID = "00000000-0000-4000-8000-000000000001"
	bearer := "Bearer " + token(profileID)

	// Write a new value.
	rr := httptest.NewRecorder()
	body := `{"profileId":"` + profileID + `","key":"attractToMainMenuFlow","data":"true"}`
	req := httptest.NewRequest(http.MethodPost,
		"/services/CustomProfileDataService/Write", strings.NewReader(body))
	req.Header.Set("Authorization", bearer)
	c.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("write status = %d, want 200", rr.Code)
	}

	// Read it back for that profile (no body on ReadAll).
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/services/CustomProfileDataService/ReadAll", nil)
	req.Header.Set("Authorization", bearer)
	c.Handler().ServeHTTP(rr, req)

	var resp customProfileReadAllResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := resp.Data["attractToMainMenuFlow"]; got != "true" {
		t.Errorf("after write, attractToMainMenuFlow = %q, want %q", got, "true")
	}
}
