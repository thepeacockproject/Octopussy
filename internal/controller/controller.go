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

		mux: http.NewServeMux(),
	}

	return c
}

// Mux returns the instantiated http.ServeMux
func (c *Controller) Mux() *http.ServeMux {
	return c.mux
}

// Handler returns the fully wrapped http.Handler, with request logging and
// panic recovery applied to every route.
func (c *Controller) Handler() http.Handler {
	return logging.HTTPMiddleware(c.Log)(c.mux)
}

// WriteJSON serializes v as JSON with the given status code.
func (c *Controller) WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		c.Log.Error("failed to encode response", slog.String("error", err.Error()))
	}
}
