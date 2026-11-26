package steam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const AppId = "3768760"

// Catalog is every known Steam entitlement (base game + DLC) for AppId.
var Catalog = []string{
	AppId, "3950810", "3950820", "3950830", "3950850", "3950870",
	"3950880", "3950900", "3950910", "3950930", "3986520", "4626160",
}

// Method selects how Steam tickets are validated, like Peacock's
// steamAuthenticationMethod.
type Method string

const (
	// MethodBackend relays the ticket to the Peacock backend.
	MethodBackend Method = "BACKEND"
	// MethodSteam calls Steam's ISteamUserAuth API directly with an API key.
	MethodSteam Method = "STEAM"
	// MethodNone skips validation and grants everyone the Catalog. Tests only.
	MethodNone Method = "NONE"
)

const (
	DefaultBackendURL = "https://backend.rdil.rocks"
	defaultSteamAPI   = "https://api.steampowered.com"
)

type Config struct {
	Method     Method
	APIKey     string
	BackendURL string
	HTTPClient *http.Client
}

// Validator authenticates Steam tickets and reports the entitlements they
// carry. It is safe for concurrent use.
type Validator struct {
	method     Method
	apiKey     string
	backendURL string
	steamAPI   string
	client     *http.Client

	mu    sync.Mutex
	cache map[string]time.Time // validated ticket hash -> expiry
}

func NewValidator(cfg Config) (*Validator, error) {
	v := &Validator{
		method:     cfg.Method,
		apiKey:     cfg.APIKey,
		backendURL: strings.TrimRight(cfg.BackendURL, "/"),
		steamAPI:   defaultSteamAPI,
		client:     cfg.HTTPClient,
		cache:      map[string]time.Time{},
	}
	if v.method == "" {
		v.method = MethodBackend
	}
	if v.client == nil {
		v.client = &http.Client{Timeout: 10 * time.Second}
	}

	switch v.method {
	case MethodBackend:
		if v.backendURL == "" {
			return nil, errors.New("steam: BACKEND auth requires a backend URL")
		}
	case MethodSteam:
		if v.apiKey == "" {
			return nil, errors.New("steam: STEAM auth requires a Steam API key")
		}
	case MethodNone:
	default:
		return nil, fmt.Errorf("steam: unknown auth method %q (want BACKEND or STEAM)", cfg.Method)
	}

	return v, nil
}

func (v *Validator) Method() Method { return v.method }

// Fallback returns the entitlements for a player with none verified: nothing,
// except under MethodNone where everyone gets the Catalog.
func (v *Validator) Fallback() []string {
	if v.method == MethodNone {
		return append([]string(nil), Catalog...)
	}
	return []string{}
}

var ErrDisabled = errors.New("steam ticket validation disabled")

// Entitlements validates the hex ticket from steam_clienttoken and returns the
// app + DLC ids it proves the player owns. steamId is the claimed
// steam_userid; identity is the optional steam_identity.
func (v *Validator) Entitlements(ctx context.Context, clientToken, identity, steamId string) ([]string, error) {
	if v.method == MethodNone {
		return nil, ErrDisabled
	}
	if clientToken == "" {
		return nil, errors.New("no steam_clienttoken supplied")
	}

	ticket, err := ParseAppTicket(clientToken)
	if err != nil {
		return nil, fmt.Errorf("parse ticket: %w", err)
	}
	if !ticket.Valid {
		return nil, errors.New("ticket signature invalid or ticket expired")
	}
	if ticket.AppId != AppId {
		return nil, fmt.Errorf("ticket is for app %s, want %s", ticket.AppId, AppId)
	}
	if ticket.SteamId != steamId {
		return nil, errors.New("ticket SteamID does not match steam_userid")
	}

	if v.cached(ticket.Hash) {
		return ticket.Entitlements(), nil
	}

	var res authResult
	if v.method == MethodBackend {
		res, err = v.fromBackend(ctx, clientToken, identity)
	} else {
		res, err = v.fromSteam(ctx, ticket, clientToken, identity)
	}
	if err != nil {
		return nil, err
	}
	if res.steamId != steamId {
		return nil, fmt.Errorf("mismatched SteamID from %s validation", v.method)
	}

	v.remember(ticket.Hash, ticket.Expiry)
	return res.entitlements, nil
}

func (v *Validator) cached(hash string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	expiry, ok := v.cache[hash]
	return ok && time.Now().Before(expiry)
}

func (v *Validator) remember(hash string, expiry time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now()
	for h, exp := range v.cache {
		if !now.Before(exp) {
			delete(v.cache, h)
		}
	}
	v.cache[hash] = expiry
}

type authResult struct {
	steamId      string
	entitlements []string
}

type backendRequest struct {
	Ticket   string `json:"ticket"`
	Identity string `json:"identity,omitempty"`
}

type backendResponse struct {
	Success      bool     `json:"success"`
	SteamId      string   `json:"steam_id"`
	Entitlements []string `json:"entitlements"`
	Error        string   `json:"error"`
}

// fromBackend calls POST /peacock/steam/validate_ticket, which answers 200 on
// success and 400 with a JSON error on rejection.
func (v *Validator) fromBackend(ctx context.Context, ticket, identity string) (authResult, error) {
	body, err := json.Marshal(backendRequest{Ticket: ticket, Identity: identity})
	if err != nil {
		return authResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		v.backendURL+"/peacock/steam/validate_ticket", bytes.NewReader(body))
	if err != nil {
		return authResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Octopussy")

	resp, err := v.client.Do(req)
	if err != nil {
		return authResult{}, fmt.Errorf("backend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return authResult{}, fmt.Errorf("backend returned %s", resp.Status)
	}

	var data backendResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return authResult{}, fmt.Errorf("backend returned %s with undecodable body: %w", resp.Status, err)
	}
	if !data.Success {
		return authResult{}, fmt.Errorf("backend rejected ticket (%s): %s", resp.Status, data.Error)
	}
	return authResult{steamId: data.SteamId, entitlements: data.Entitlements}, nil
}

type steamAuthResponse struct {
	Response struct {
		Error *struct {
			ErrorCode int    `json:"errorcode"`
			ErrorDesc string `json:"errordesc"`
		} `json:"error"`
		Params *struct {
			Result  string `json:"result"`
			SteamId string `json:"steamid"`
		} `json:"params"`
	} `json:"response"`
}

// fromSteam calls ISteamUserAuth/AuthenticateUserTicket. Steam only confirms
// who owns the ticket; the entitlements come from the ticket itself.
func (v *Validator) fromSteam(ctx context.Context, parsed *AppTicket, ticket, identity string) (authResult, error) {
	q := url.Values{}
	q.Set("key", v.apiKey)
	q.Set("appid", parsed.AppId)
	q.Set("ticket", ticket)
	if identity != "" {
		q.Set("identity", identity)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		v.steamAPI+"/ISteamUserAuth/AuthenticateUserTicket/v1?"+q.Encode(), nil)
	if err != nil {
		return authResult{}, err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		// Not wrapped: *url.Error would leak the API key from the query.
		return authResult{}, errors.New("steam request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return authResult{}, fmt.Errorf("steam returned %s", resp.Status)
	}

	var data steamAuthResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return authResult{}, fmt.Errorf("decode steam response: %w", err)
	}
	if e := data.Response.Error; e != nil {
		return authResult{}, fmt.Errorf("steam error %d: %s", e.ErrorCode, e.ErrorDesc)
	}
	if p := data.Response.Params; p == nil || p.Result != "OK" {
		return authResult{}, errors.New("steam did not return result OK")
	}

	return authResult{
		steamId:      data.Response.Params.SteamId,
		entitlements: parsed.Entitlements(),
	}, nil
}
