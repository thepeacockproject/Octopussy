package api

import (
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/base"
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/other"
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/services"
	"github.com/thepeacockproject/Octopussy/internal/controller"
)

func RegisterRoutes(c *controller.Controller) {
	base.NewHandler(c).RegisterRoutes()
	other.NewHandler(c).RegisterRoutes()
	services.NewHandler(c).RegisterRoutes()
}
