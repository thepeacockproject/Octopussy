package services

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type getBlobOfflineCacheDatabaseDiffRequest struct {
	ProductVersion string `json:"productVersion"`
}

func (c *Handler) handleGetBlobOfflineCacheDatabaseDiff(w http.ResponseWriter, r *http.Request) {
	var req getBlobOfflineCacheDatabaseDiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	c.WriteJSON(w, http.StatusOK, fmt.Sprintf("Unknown product version %s", req.ProductVersion))
}
