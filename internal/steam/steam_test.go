package steam

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testSteamId = "76561197960287930"

// testKey stands in for Valve's key; tickets carry a 1024-bit signature.
var testKey = func() *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return k
}()

func useTestKey(t *testing.T) {
	t.Helper()
	orig := steamPublicKey
	steamPublicKey = &testKey.PublicKey
	t.Cleanup(func() { steamPublicKey = orig })
}

type ticketSpec struct {
	steamId string
	appId   uint32
	expiry  time.Time
	dlc     []uint32
	version uint32
}

func defaultSpec() ticketSpec {
	return ticketSpec{
		steamId: testSteamId,
		appId:   3768760,
		expiry:  time.Now().Add(24 * time.Hour),
		dlc:     []uint32{3950810, 3950820},
		version: 4,
	}
}

// buildTicket encodes a GC token prefix, signed ownership ticket and signature.
func buildTicket(t *testing.T, s ticketSpec) []byte {
	t.Helper()
	le := binary.LittleEndian

	var own []byte
	own = le.AppendUint32(own, 0) // length, patched below
	own = le.AppendUint32(own, s.version)
	sid, err := strconv.ParseUint(s.steamId, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	own = le.AppendUint64(own, sid)
	own = le.AppendUint32(own, s.appId)
	own = append(own, make([]byte, 16)...) // IPs, flags, time issued
	own = le.AppendUint32(own, uint32(s.expiry.Unix()))
	own = le.AppendUint16(own, 1) // one license...
	own = le.AppendUint32(own, 42)
	own = le.AppendUint16(own, uint16(len(s.dlc)))
	for _, d := range s.dlc {
		own = le.AppendUint32(own, d)
		own = le.AppendUint16(own, 0) // no DLC licenses
	}
	own = le.AppendUint16(own, 0) // reserved
	le.PutUint32(own, uint32(len(own)))

	digest := sha1.Sum(own)
	sig, err := rsa.SignPKCS1v15(rand.Reader, testKey, crypto.SHA1, digest[:])
	if err != nil {
		t.Fatal(err)
	}

	prefix := le.AppendUint32(nil, 20)
	prefix = append(prefix, make([]byte, 52)...)

	out := append(prefix, own...)
	return append(out, sig...)
}

func ticketHex(t *testing.T, s ticketSpec) string {
	return strings.ToUpper(hex.EncodeToString(buildTicket(t, s)))
}

func TestParseAppTicket(t *testing.T) {
	useTestKey(t)
	spec := defaultSpec()

	tk, err := ParseAppTicket(ticketHex(t, spec))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !tk.Valid {
		t.Error("Valid = false, want true")
	}
	if tk.SteamId != testSteamId || tk.AppId != AppId {
		t.Errorf("SteamId/AppId = %s/%s", tk.SteamId, tk.AppId)
	}
	if got := strings.Join(tk.Entitlements(), ","); got != AppId+",3950810,3950820" {
		t.Errorf("Entitlements = %s", got)
	}
	if len(tk.Hash) != 64 {
		t.Errorf("Hash = %q, want 64 hex chars", tk.Hash)
	}
}

func TestParseAppTicketRejectsTampering(t *testing.T) {
	useTestKey(t)
	raw := buildTicket(t, defaultSpec())
	raw[56+16] ^= 0xFF // inside the signed app id

	tk, err := parseAppTicket(raw, steamPublicKey, time.Now())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tk.Valid {
		t.Error("tampered ticket reported valid")
	}
}

func TestParseAppTicketRejectsExpired(t *testing.T) {
	useTestKey(t)
	spec := defaultSpec()
	spec.expiry = time.Now().Add(-time.Hour)

	tk, err := ParseAppTicket(ticketHex(t, spec))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tk.Valid {
		t.Error("expired ticket reported valid")
	}
}

func TestParseAppTicketMalformed(t *testing.T) {
	useTestKey(t)
	full := buildTicket(t, defaultSpec())

	badVersion := defaultSpec()
	badVersion.version = 3

	cases := map[string]string{
		"not hex":     "zz",
		"empty":       "",
		"truncated":   hex.EncodeToString(full[:70]),
		"bad version": ticketHex(t, badVersion),
	}
	for name, in := range cases {
		if _, err := ParseAppTicket(in); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func newBackendValidator(t *testing.T, h http.HandlerFunc) (*Validator, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)

	v, err := NewValidator(Config{Method: MethodBackend, BackendURL: srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	return v, &calls
}

func TestBackendValidationAndCache(t *testing.T) {
	useTestKey(t)
	ticket := ticketHex(t, defaultSpec())

	v, calls := newBackendValidator(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/peacock/steam/validate_ticket" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["ticket"] != ticket {
			t.Errorf("ticket not forwarded verbatim")
		}
		if body["identity"] != "octo" {
			t.Errorf("identity = %q, want octo", body["identity"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "steam_id": testSteamId,
			"entitlements": []string{AppId, "3950810", "3950820"},
		})
	})

	for i := range 2 {
		ents, err := v.Entitlements(context.Background(), ticket, "octo", testSteamId)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if len(ents) != 3 || ents[0] != AppId {
			t.Errorf("call %d: entitlements = %v", i, ents)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("backend called %d times, want 1 (second should hit cache)", n)
	}
}

func TestBackendRejection(t *testing.T) {
	useTestKey(t)
	v, _ := newBackendValidator(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false,"error":"Ticket failed validation."}`))
	})

	_, err := v.Entitlements(context.Background(), ticketHex(t, defaultSpec()), "", testSteamId)
	if err == nil || !strings.Contains(err.Error(), "Ticket failed validation.") {
		t.Fatalf("err = %v, want backend rejection", err)
	}
}

func TestBackendSteamIdMismatch(t *testing.T) {
	useTestKey(t)
	v, _ := newBackendValidator(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"steam_id":"1","entitlements":["3768760"]}`))
	})

	if _, err := v.Entitlements(context.Background(), ticketHex(t, defaultSpec()), "", testSteamId); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestLocalChecksRunBeforeRemote(t *testing.T) {
	useTestKey(t)
	v, calls := newBackendValidator(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("backend should not be called")
	})

	wrongApp := defaultSpec()
	wrongApp.appId = 1659040
	expired := defaultSpec()
	expired.expiry = time.Now().Add(-time.Minute)

	cases := map[string]struct{ ticket, steamId string }{
		"no ticket":      {"", testSteamId},
		"wrong app":      {ticketHex(t, wrongApp), testSteamId},
		"expired":        {ticketHex(t, expired), testSteamId},
		"other steam id": {ticketHex(t, defaultSpec()), "76561197960287931"},
	}
	for name, tc := range cases {
		if _, err := v.Entitlements(context.Background(), tc.ticket, "", tc.steamId); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if calls.Load() != 0 {
		t.Errorf("backend called %d times", calls.Load())
	}
}

func newSteamValidator(t *testing.T, method Method, h http.HandlerFunc) *Validator {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	v, err := NewValidator(Config{Method: method, APIKey: "secret-key"})
	if err != nil {
		t.Fatal(err)
	}
	v.steamAPI = srv.URL
	return v
}

func TestSteamValidation(t *testing.T) {
	useTestKey(t)
	ticket := ticketHex(t, defaultSpec())

	v := newSteamValidator(t, MethodSteam, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/ISteamUserAuth/AuthenticateUserTicket/v1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if q.Get("key") != "secret-key" || q.Get("appid") != AppId || q.Get("ticket") != ticket {
			t.Errorf("query = %v", q)
		}
		if q.Has("identity") {
			t.Error("identity should be omitted when empty")
		}
		_, _ = w.Write([]byte(`{"response":{"params":{"result":"OK","steamid":"` + testSteamId + `"}}}`))
	})

	ents, err := v.Entitlements(context.Background(), ticket, "", testSteamId)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ents, ","); got != AppId+",3950810,3950820" {
		t.Errorf("entitlements = %s", got)
	}
}

func TestSteamError(t *testing.T) {
	useTestKey(t)
	v := newSteamValidator(t, MethodSteam, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"error":{"errorcode":101,"errordesc":"Invalid ticket"}}}`))
	})

	_, err := v.Entitlements(context.Background(), ticketHex(t, defaultSpec()), "", testSteamId)
	if err == nil || !strings.Contains(err.Error(), "101") {
		t.Fatalf("err = %v, want steam error 101", err)
	}
}

func TestValidatorConfig(t *testing.T) {
	for _, cfg := range []Config{
		{Method: "OFFICIAL"},
		{Method: MethodSteam},
		{Method: MethodBackend},
	} {
		if _, err := NewValidator(cfg); err == nil {
			t.Errorf("%+v: expected config error", cfg)
		}
	}

	def, err := NewValidator(Config{BackendURL: DefaultBackendURL})
	if err != nil || def.Method() != MethodBackend {
		t.Errorf("default method = %v (err %v), want BACKEND", def.Method(), err)
	}
	if fb := def.Fallback(); fb == nil || len(fb) != 0 {
		t.Errorf("fallback = %v, want empty non-nil", fb)
	}

	none, _ := NewValidator(Config{Method: MethodNone})
	if _, err := none.Entitlements(context.Background(), "00", "", "1"); !errors.Is(err, ErrDisabled) {
		t.Errorf("err = %v, want ErrDisabled", err)
	}
	if len(none.Fallback()) != len(Catalog) {
		t.Error("NONE fallback should be the catalog")
	}
}
