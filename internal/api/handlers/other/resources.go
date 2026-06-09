package other

import (
	"fmt"
	"net/http"
)

func (c *Handler) handleResources(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")

	result := fmt.Sprintf(""+
		"<?xml version=\"1.0\" encoding=\"utf-8\"?>"+
		"<Error>"+
		"<Code>BlobNotFound</Code>"+
		"<Message>The specified blob does not exist: %s</Message>"+
		"</Error>",
		path,
	)

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(result))
}
