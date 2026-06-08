// Package controller wires the server's services together and exposes the
// HTTP routing surface. It plays the same central role as Peacock's Controller:
// a single object that owns shared services (here, the logger and JWT manager)
// and registers the routes that use them.
package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/content"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
	"github.com/thepeacockproject/Octopussy/internal/logging"
	"github.com/thepeacockproject/Octopussy/internal/store"
)

// Controller holds the shared services and the configured router.
type Controller struct {
	Log     *slog.Logger
	JWT     *jwt.Manager
	Store   *store.Store
	Content *content.Store

	mux *http.ServeMux
}

// New builds a Controller and registers its routes.
func New(log *slog.Logger, jwtMgr *jwt.Manager, st *store.Store, gameContent *content.Store) *Controller {
	c := &Controller{
		Log:     log,
		JWT:     jwtMgr,
		Store:   st,
		Content: gameContent,
		mux:     http.NewServeMux(),
	}
	c.registerRoutes()
	return c
}

// Handler returns the fully wrapped http.Handler, with request logging and
// panic recovery applied to every route.
func (c *Controller) Handler() http.Handler {
	return logging.HTTPMiddleware(c.Log)(c.mux)
}

// registerRoutes attaches every route to the mux. Protected routes are wrapped
// in the JWT middleware so they only run with a valid bearer token.
func (c *Controller) registerRoutes() {
	c.mux.HandleFunc("GET /health", c.handleHealth)
	c.mux.HandleFunc("POST /oauth/token", c.handleOAuthToken)

	// Config discovery: the game's first call, telling it where services live.
	c.mux.HandleFunc("GET /config/{configName}/{version}", c.handleOnlineConfig)

	// Static documents the config discovery doc points at.
	c.mux.HandleFunc("GET /files/onlineconfig.json", c.handleOnlineConfigFile)

	// TODO: Add some sort of wildcard support for "_pc_en"
	c.mux.HandleFunc("HEAD /files/privacypolicy/knt/privacypolicy_pc_en.json", c.handlePrivacyPolicy)
	c.mux.HandleFunc("GET /files/privacypolicy/knt/privacypolicy_pc_en.json", c.handlePrivacyPolicy)

	// Resources
	// TODO: Add some sort of wildcard support for the version in the URL
	c.mux.HandleFunc("GET /resources-1-0/{path...}", c.handleResources)
	c.mux.HandleFunc("GET /resources-1-1/{path...}", c.handleResources)

	// Metrics ingest: a no-op telemetry sink (config discovery points here).
	c.mux.HandleFunc("POST /ingest", c.handleMetricsIngest)

	// Authentication configuration: the game's post-login bootstrap call.
	c.mux.Handle("GET /authentication/api/configuration/Init",
		c.JWT.Middleware(http.HandlerFunc(c.handleAuthConfigInit)))

	// AuthenticationService
	c.mux.Handle("POST services/AuthenticationService/GetBlobOfflineCacheDatabaseDiff",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetBlobOfflineCacheDatabaseDiff)))

	// KntAggregateService: serves the aggregated menu content (mission catalog).
	c.mux.Handle("POST /services/KntAggregateService/GetMenuContent",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetMenuContent)))

	// CustomProfileDataService: persistent per-profile key/value store. Both run
	// behind the JWT middleware so the profile id is taken from the bearer token.
	c.mux.Handle("POST /services/CustomProfileDataService/ReadAll",
		c.JWT.Middleware(http.HandlerFunc(c.handleCustomProfileReadAll)))
	c.mux.Handle("POST /services/CustomProfileDataService/Write",
		c.JWT.Middleware(http.HandlerFunc(c.handleCustomProfileWrite)))

	// SharedProfileService: public-facing profile lookup (gamertag, linked
	// accounts) for one or more users.
	c.mux.Handle("POST /services/SharedProfileService/ResolveProfiles",
		c.JWT.Middleware(http.HandlerFunc(c.handleResolveProfiles)))

	// KntProfileService: Entitlements & player progression.
	c.mux.Handle("POST /services/KntProfileService/SetClientEntitlements",
		c.JWT.Middleware(http.HandlerFunc(c.handleSetClientEntitlements)))
	c.mux.Handle("POST /services/KntProfileService/GetEntitlements",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetEntitlements)))
	c.mux.Handle("POST /services/KntProfileService/GetPlayerProgression",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetPlayerProgression)))

	// KntProgressionService: challenges.
	c.mux.Handle("POST /services/KntProgressionService/GetChallenges",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetChallenges)))
	c.mux.Handle("POST /services/KntProgressionService/GetChallengeMosaic",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))

	// KntSessionService: sessions.
	c.mux.Handle("POST /services/KntSessionService/GetForPlay",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetForPlay)))
	c.mux.Handle("POST /services/KntSessionService/StopSession",
		c.JWT.Middleware(http.HandlerFunc(c.handleStopSession)))
	c.mux.Handle("POST /services/KntSessionService/CompleteActiveChallenge",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))

	// KntInventoryService
	c.mux.Handle("POST /services/KntInventoryService/GetInventory",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))
	c.mux.Handle("POST /services/KntInventoryService/GetUnlockables",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))
	c.mux.Handle("POST /services/KntInventoryService/AcquireUnlockable",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))

	// KntLeaderboardService
	c.mux.Handle("POST /services/KntLeaderboardService/GetLeaderboards",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))
	c.mux.Handle("POST /services/KntLeaderboardService/GetFriendLeaderboards",
		c.JWT.Middleware(http.HandlerFunc(c.handleHealth)))
}

// requestBaseURL reconstructs the scheme://host this server was reached at,
// honoring reverse-proxy headers. It is used to point config responses back at
// Octopussy itself.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	return scheme + "://" + host
}

// writeJSON serializes v as JSON with the given status code.
func (c *Controller) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		c.Log.Error("failed to encode response", slog.String("error", err.Error()))
	}
}

// handleHealth is a simple liveness probe.
func (c *Controller) handleHealth(w http.ResponseWriter, _ *http.Request) {
	c.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
