package other

import (
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/controller"
)

type Handler struct {
	*controller.Controller
}

func NewHandler(controller *controller.Controller) *Handler {
	return &Handler{
		controller,
	}
}

func (c *Handler) RegisterRoutes() {
	mux := c.Mux()

	mux.HandleFunc("POST /oauth/token", c.handleOAuthToken)

	// Config discovery: the game's first call, telling it where services live.
	mux.HandleFunc("GET /config/{configName}/{version}", c.handleOnlineConfig)

	// Static documents the config discovery doc points at.
	mux.HandleFunc("GET /files/onlineconfig.json", c.handleOnlineConfigFile)

	// TODO: Add some sort of wildcard support for "_pc_en"
	mux.HandleFunc("HEAD /files/privacypolicy/knt/privacypolicy_pc_en.json", c.handlePrivacyPolicy)
	mux.HandleFunc("GET /files/privacypolicy/knt/privacypolicy_pc_en.json", c.handlePrivacyPolicy)

	// Resources
	// TODO: Add some sort of wildcard support for the version in the URL
	mux.HandleFunc("GET /resources-1-0/{path...}", c.handleResources)
	mux.HandleFunc("GET /resources-1-1/{path...}", c.handleResources)

	// Metrics ingest: a no-op telemetry sink (config discovery points here).
	mux.HandleFunc("POST /ingest", c.handleMetricsIngest)

	// Authentication configuration: the game's post-login bootstrap call.
	mux.Handle("GET /authentication/api/configuration/Init",
		c.JWT.Middleware(http.HandlerFunc(c.handleAuthConfigInit)))
}
