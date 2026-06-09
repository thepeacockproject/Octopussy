package base

import (
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
	
	mux.HandleFunc("GET /health", c.handleHealth)
}
