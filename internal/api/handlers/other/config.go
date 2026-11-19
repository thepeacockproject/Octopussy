package other

import (
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
)

// supportedGameVersion is the GAME_VER our server reports to clients. The game
// sends the client's own version in the request path; this is the version the
// server tells it to run as.
const supportedGameVersion = "1.1.6"

// Online config response models. These mirror the JSON the IOI "kconf" config
// discovery service returns. The endpoint tells the game which backend services
// to talk to; since Octopussy is the only server, every URL points at us.

type onlineConfigResponse struct {
	Versions []onlineConfigVersion `json:"Versions"`
}

type onlineConfigVersion struct {
	Name      string                `json:"Name"`
	GameVer   string                `json:"GAME_VER"`
	ServerVer onlineConfigServerVer `json:"SERVER_VER"`
	IssuerId  string                `json:"ISSUER_ID"`
}

type onlineConfigServerVer struct {
	Metrics              onlineConfigMetrics       `json:"Metrics"`
	Authentication       onlineConfigAuth          `json:"Authentication"`
	Configuration        onlineConfigConfiguration `json:"Configuration"`
	Resources            onlineConfigResources     `json:"Resources"`
	GlobalAuthentication onlineConfigGlobalAuth    `json:"GlobalAuthentication"`
}

type onlineConfigMetrics struct {
	MetricsServerHost        string `json:"MetricsServerHost"`
	MetricsServerServicePath string `json:"MetricsServerServicePath"`
	MetricsRouterBaseUrl     string `json:"MetricsRouterBaseUrl"`
	MetricsRouterNamespace   string `json:"MetricsRouterNamespace"`
}

type onlineConfigAuth struct {
	AuthenticationHost                     string `json:"AuthenticationHost"`
	AuthenticationServicePathConfiguration string `json:"AuthenticationServicePathConfiguration"`
	UserChannelServicePath                 string `json:"UserChannelServicePath"`
	PingPath                               string `json:"PingPath"`
}

type onlineConfigConfiguration struct {
	Url          string `json:"Url"`
	AgreementUrl string `json:"AgreementUrl"`
}

type onlineConfigResources struct {
	ResourcesServicePath string `json:"ResourcesServicePath"`
}

type onlineConfigGlobalAuth struct {
	AuthenticationHost string `json:"AuthenticationHost"`
	OAuthPath          string `json:"OAuthPath"`
	RequestedAudience  string `json:"RequestedAudience"`
}

// handleOnlineConfig serves the config discovery document. The game hits this
// first to learn where every other service lives; we answer with our own host
// for all of them so the client routes everything back to Octopussy.
func (c *Handler) handleOnlineConfig(w http.ResponseWriter, r *http.Request) {
	configName := r.PathValue("configName")
	version := r.PathValue("version")
	issuer := r.URL.Query().Get("issuer")

	base := shared.RequestBaseURL(r)

	resp := onlineConfigResponse{
		Versions: []onlineConfigVersion{{
			Name:     configName,
			GameVer:  supportedGameVersion,
			IssuerId: issuer,
			ServerVer: onlineConfigServerVer{
				Metrics: onlineConfigMetrics{
					MetricsServerHost:        base + "/ingest",
					MetricsServerServicePath: "",
					MetricsRouterBaseUrl:     base + "/ingest",
					MetricsRouterNamespace:   "knt",
				},
				Authentication: onlineConfigAuth{
					AuthenticationHost:                     base,
					AuthenticationServicePathConfiguration: "authentication/api/configuration",
					UserChannelServicePath:                 "services",
					PingPath:                               "/ping",
				},
				Configuration: onlineConfigConfiguration{
					Url:          base + "/files/onlineconfig.json",
					AgreementUrl: base + "/files/privacypolicy/knt/privacypolicy.json",
				},
				Resources: onlineConfigResources{
					ResourcesServicePath: base + "/resources-1-0",
				},
				GlobalAuthentication: onlineConfigGlobalAuth{
					AuthenticationHost: base,
					OAuthPath:          "oauth/token",
					RequestedAudience:  "steam-prod_1",
				},
			},
		}},
	}

	c.Log.Debug("served online config",
		slog.String("config", configName),
		slog.String("version", version),
		slog.String("issuer", issuer),
	)
	c.WriteJSON(w, http.StatusOK, resp)
}
