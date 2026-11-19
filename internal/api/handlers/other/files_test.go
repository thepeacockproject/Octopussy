package other_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/other"
)

func TestOnlineConfigFile(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/onlineconfig.json", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp other.OnlineConfigFile
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	o := resp.Online
	if o.WebSocketsEnabled {
		t.Error("WebSocketsEnabled = true, want false")
	}
}

func TestPrivacyPolicyFile(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/privacypolicy/knt/privacypolicy_pc_en.json", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if v := rr.Header().Get("x-ms-meta-version"); v != "20260529" {
		t.Errorf("x-ms-meta-version = %q, want 20260529", v)
	}

	// The body is served verbatim and must remain valid JSON.
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(doc["view"]) != `"menu3.modal.ModalDialogGeneric"` {
		t.Errorf("view = %s, want the modal dialog view", doc["view"])
	}
}

func TestPrivacyPolicyVariants(t *testing.T) {
	c, _ := apitest.NewTestController(t)
	want := `{"view":"menu3.modal.ModalDialogGeneric","type":"gdprconsent"}`

	for _, name := range []string{"privacypolicy_pc_en.json", "privacypolicy_en.json"} {
		t.Run("HEAD "+name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodHead, "/files/privacypolicy/knt/"+name, nil)
			c.Handler().ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			if cl := rr.Header().Get("Content-Length"); cl != strconv.Itoa(len(want)) {
				t.Errorf("Content-Length = %q, want %d", cl, len(want))
			}
			if rr.Body.Len() != 0 {
				t.Errorf("HEAD returned a %d-byte body", rr.Body.Len())
			}
		})
	}

	t.Run("unrelated name 404s", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/files/privacypolicy/knt/other.json", nil)
		c.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rr.Code)
		}
	})
}
