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
	"github.com/thepeacockproject/Octopussy/internal/persistence"
	"github.com/thepeacockproject/Octopussy/internal/steam"
)

// Controller holds the shared services and the configured router.
type Controller struct {
	Log         *slog.Logger
	JWT         *jwt.Manager
	Persistence *persistence.Store
	Content     *content.Store
	Steam       *steam.Validator

	mux *http.ServeMux
}

type middleware func(http.Handler) http.Handler

// New builds a Controller and registers its routes.
func New(log *slog.Logger, jwtMgr *jwt.Manager, st *persistence.Store, gameContent *content.Store, steamValidator *steam.Validator) *Controller {
	c := &Controller{
		Log:         log,
		JWT:         jwtMgr,
		Persistence: st,
		Content:     gameContent,
		Steam:       steamValidator,

		mux: http.NewServeMux(),
	}

	return c
}

// Mux returns the instantiated http.ServeMux
func (c *Controller) Mux() *http.ServeMux {
	return c.mux
}

func (c *Controller) Handler(middlewares ...middleware) http.Handler {
	var handler http.Handler = c.mux

	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] == nil {
			continue
		}

		handler = middlewares[i](handler)
	}

	return handler
}

// WriteJSON serializes v as JSON with the given status code.
func (c *Controller) WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		c.Log.Error("failed to encode response", slog.String("error", err.Error()))
	}
}
