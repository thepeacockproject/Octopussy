package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

type Route struct {
	Prefix      string
	Target      string
	KeepPrefix  bool
	RewriteBody func(path, body, base string) string
}

type ctxKey int

const matchedKey ctxKey = iota

type matched struct {
	route        *Route
	base         string
	incomingPath string
}

func middleware(log *slog.Logger, localBase string, routes []Route) func(http.Handler) http.Handler {
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			m, _ := pr.In.Context().Value(matchedKey).(*matched)
			if m == nil {
				return
			}

			target, err := url.Parse(m.route.Target)
			if err != nil {
				return
			}

			path := pr.In.URL.Path
			if !m.route.KeepPrefix {
				path = strings.TrimPrefix(path, strings.TrimSuffix(m.route.Prefix, "/"))
			}

			pr.Out.URL.Scheme = target.Scheme
			pr.Out.URL.Host = target.Host
			pr.Out.URL.Path = strings.TrimSuffix(target.Path, "/") + path
			pr.Out.Host = target.Host
		},
		ModifyResponse: func(resp *http.Response) error {
			body, err := readBody(resp)
			if err != nil {
				return err
			}

			if m, _ := resp.Request.Context().Value(matchedKey).(*matched); m != nil && m.route.RewriteBody != nil {
				body = []byte(m.route.RewriteBody(m.incomingPath, string(body), m.base))
			}

			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
			resp.Header.Set("Content-Length", strconv.Itoa(len(body)))

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("reverse proxy: upstream request failed",
				slog.String("url", r.URL.String()),
				slog.String("error", err.Error()),
			)

			w.WriteHeader(http.StatusBadGateway)
		},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := matchRoute(routes, r.URL.Path)
			if route == nil {
				log.Warn("reverse proxy: no upstream, passing through", slog.String("path", r.URL.Path))
				next.ServeHTTP(w, r)
				return
			}

			m := &matched{route: route, base: localBase, incomingPath: r.URL.Path}
			rp.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), matchedKey, m)))
		})
	}
}

func readBody(resp *http.Response) ([]byte, error) {
	src := resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}

		defer gz.Close()
		src = gz

		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Transfer-Encoding")
	}

	defer resp.Body.Close()

	return io.ReadAll(src)
}

func matchRoute(routes []Route, path string) *Route {
	var best *Route

	for i := range routes {
		if strings.HasPrefix(path, routes[i].Prefix) {
			if best == nil || len(routes[i].Prefix) > len(best.Prefix) {
				best = &routes[i]
			}
		}
	}

	return best
}
