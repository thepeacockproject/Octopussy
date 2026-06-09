package services

import (
	"fmt"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
)

type getBlobOfflineCacheDatabaseDiffRequest struct {
	ProductVersion string `json:"productVersion"`
}

func (c *Handler) handleGetBlobOfflineCacheDatabaseDiff(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[getBlobOfflineCacheDatabaseDiffRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c.WriteJSON(w, http.StatusOK, fmt.Sprintf("Unknown product version %s", body.ProductVersion))
}
