package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOnlineConfigDiscovery(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config/pc-prod/1_1_7?issuer=3768760", nil)
	req.Host = "octopussy.example:8080"
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp onlineConfigResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Versions) != 1 {
		t.Fatalf("got %d versions, want 1", len(resp.Versions))
	}

	v := resp.Versions[0]
	// Name and ISSUER_ID are echoed from the request path/query.
	if v.Name != "pc-prod" {
		t.Errorf("Name = %q, want pc-prod", v.Name)
	}
	if v.IssuerID != "3768760" {
		t.Errorf("ISSUER_ID = %q, want 3768760", v.IssuerID)
	}
	if v.GameVer != supportedGameVersion {
		t.Errorf("GAME_VER = %q, want %q", v.GameVer, supportedGameVersion)
	}

	// Every service URL should point back at the requesting host.
	const want = "http://octopussy.example:8080"
	sv := v.ServerVer
	urls := map[string]string{
		"Authentication.AuthenticationHost":       sv.Authentication.AuthenticationHost,
		"Metrics.MetricsServerHost":               sv.Metrics.MetricsServerHost,
		"Metrics.MetricsRouterBaseUrl":            sv.Metrics.MetricsRouterBaseUrl,
		"Configuration.Url":                       sv.Configuration.Url,
		"Configuration.AgreementUrl":              sv.Configuration.AgreementUrl,
		"Resources.ResourcesServicePath":          sv.Resources.ResourcesServicePath,
		"GlobalAuthentication.AuthenticationHost": sv.GlobalAuthentication.AuthenticationHost,
	}
	for name, got := range urls {
		if !strings.HasPrefix(got, want) {
			t.Errorf("%s = %q, want prefix %q", name, got, want)
		}
	}
}

func TestOnlineConfigHonorsForwardedProto(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config/pc-prod/1_1_7?issuer=1", nil)
	req.Host = "bond.example"
	req.Header.Set("X-Forwarded-Proto", "https")
	c.Handler().ServeHTTP(rr, req)

	var resp onlineConfigResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := resp.Versions[0].ServerVer.GlobalAuthentication.AuthenticationHost; got != "https://bond.example" {
		t.Errorf("AuthenticationHost = %q, want https://bond.example", got)
	}
}
