package other_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/other"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
)

func TestAuthConfigInitRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authentication/api/configuration/Init?configName=pc-prod&gamerTag=iamrdil", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestAuthConfigInit(t *testing.T) {
	c, _ := apitest.NewTestController(t)
	userId := uuid.Nil.String()

	// Mint a token with the full claim set Init derives its fields from.
	now := time.Now()
	tok, err := c.JWT.Sign(jwt.Claims{
		Platform: "steam",
		UserId:   "76561190000000000",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userId,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)),
		},
	})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/authentication/api/configuration/Init?configName=pc-prod&gamerTag=iamrdil", nil)
	req.Host = "bond.example"
	req.Header.Set("Authorization", "Bearer "+tok)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp other.AuthConfigInitResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// profileid mirrors the token subject.
	if resp.ProfileId != userId {
		t.Errorf("profileid = %q, want %q", resp.ProfileId, userId)
	}

	// token = "{exp}-{iat}-{platform}-{userid}".
	wantToken := strings.Join([]string{
		strconv.FormatInt(now.Add(2*time.Hour).Unix(), 10),
		strconv.FormatInt(now.Unix(), 10),
		"steam",
		"76561190000000000",
	}, "-")
	if resp.Token != wantToken {
		t.Errorf("token = %q, want %q", resp.Token, wantToken)
	}

	// Blob URL points back at the requesting host.
	if !strings.HasPrefix(resp.BlobConfig.BlobURL, "http://bond.example/") {
		t.Errorf("bloburl = %q, want prefix http://bond.example/", resp.BlobConfig.BlobURL)
	}
	if resp.BlobConfig.BlobSigDuration != 7200000 {
		t.Errorf("blobsigduration = %v, want 7200000", resp.BlobConfig.BlobSigDuration)
	}
	if resp.ServerVersion != other.AuthServerVersion {
		t.Errorf("serverversion = %q, want %q", resp.ServerVersion, other.AuthServerVersion)
	}

	// servertimeutc must parse as the .NET-style UTC stamp we emit.
	if _, err := time.Parse(other.DotNetTimeLayout+"Z", resp.ServerTimeUTC); err != nil {
		t.Errorf("servertimeutc %q not in expected layout: %v", resp.ServerTimeUTC, err)
	}
}
