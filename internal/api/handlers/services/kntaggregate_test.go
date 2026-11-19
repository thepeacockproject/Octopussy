package services_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
	"github.com/thepeacockproject/Octopussy/internal/content"
)

func TestGetMenuContentRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/KntAggregateService/GetMenuContent", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestGetMenuContent(t *testing.T) {
	c, token := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/KntAggregateService/GetMenuContent", nil)
	req.Header.Set("Authorization", "Bearer "+token(uuid.Nil.String()))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp content.MenuContent
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.MissionLists) != 1 || resp.MissionLists[0].Name != "KNT_Missions" {
		t.Errorf("unexpected mission lists: %+v", resp.MissionLists)
	}
	if len(resp.Missions) != 1 {
		t.Errorf("got %d missions, want 1", len(resp.Missions))
	}
}
