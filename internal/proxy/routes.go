package proxy

import (
	"log/slog"
	"net/http"
	"strings"
)

var routes []Route

func init() {
	routes = []Route{
		{Prefix: "/config", Target: "https://kconf.ioi-cloud.com", KeepPrefix: true, RewriteBody: rewriteConfigBody},
		{Prefix: "/knteventsprod/", Target: "https://knteventsprod.ioi-cloud.com"},
		{Prefix: "/iokntprod01/", Target: "https://iokntprod01.ioi-cloud.com"},
		{Prefix: "/iokntprod01storage/", Target: "https://iokntprod01storage.blob.core.windows.net"},
		{Prefix: "/kntprodconfigstorage/", Target: "https://kntprodconfigstorage.blob.core.windows.net"},
		{Prefix: "/kauth/", Target: "https://kauth.ioi-cloud.com", RewriteBody: rewriteKauthBody},
	}
}

func HTTPMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return middleware(log, "http://localhost", routes)
}

func rewriteConfigBody(_ string, body, base string) string {
	for _, r := range routes {
		if r.KeepPrefix {
			continue
		}

		body = strings.ReplaceAll(body, r.Target, base+strings.TrimSuffix(r.Prefix, "/"))
	}

	return body
}

func rewriteKauthBody(path, body, base string) string {
	if !strings.Contains(path, "/authentication/api/configuration/Init") {
		return body
	}

	return strings.ReplaceAll(body, "https://iokntprod01storage.blob.core.windows.net", base+"/iokntprod01storage")
}
