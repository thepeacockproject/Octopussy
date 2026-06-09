package services

import (
	"net/http"

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

	// AuthenticationService
	mux.Handle("POST /services/AuthenticationService/GetBlobOfflineCacheDatabaseDiff",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetBlobOfflineCacheDatabaseDiff)))

	// KntAggregateService: serves the aggregated menu content (mission catalog).
	mux.Handle("POST /services/KntAggregateService/GetMenuContent",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetMenuContent)))

	// CustomProfileDataService: persistent per-profile key/value store. Both run
	// behind the JWT middleware so the profile id is taken from the bearer token.
	mux.Handle("POST /services/CustomProfileDataService/ReadAll",
		c.JWT.Middleware(http.HandlerFunc(c.handleCustomProfileReadAll)))
	mux.Handle("POST /services/CustomProfileDataService/Write",
		c.JWT.Middleware(http.HandlerFunc(c.handleCustomProfileWrite)))

	// SharedProfileService: public-facing profile lookup (gamertag, linked
	// accounts) for one or more users.
	mux.Handle("POST /services/SharedProfileService/ResolveProfiles",
		c.JWT.Middleware(http.HandlerFunc(c.handleResolveProfiles)))

	// KntProfileService: Entitlements & player progression.
	mux.Handle("POST /services/KntProfileService/SetClientEntitlements",
		c.JWT.Middleware(http.HandlerFunc(c.handleSetClientEntitlements)))
	mux.Handle("POST /services/KntProfileService/GetEntitlements",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetEntitlements)))
	mux.Handle("POST /services/KntProfileService/GetPlayerProgression",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetPlayerProgression)))

	// KntProgressionService: challenges.
	mux.Handle("POST /services/KntProgressionService/GetChallenges",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetChallenges)))
	mux.Handle("POST /services/KntProgressionService/GetChallengeMosaic",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetChallengeMosaic)))

	// KntSessionService: sessions.
	mux.Handle("POST /services/KntSessionService/GetForPlay",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetForPlay)))
	mux.Handle("POST /services/KntSessionService/StopSession",
		c.JWT.Middleware(http.HandlerFunc(c.handleStopSession)))
	mux.Handle("POST /services/KntSessionService/CompleteActiveChallenge",
		c.JWT.Middleware(http.HandlerFunc(c.handleCompleteActiveChallenge)))

	// KntInventoryService
	mux.Handle("POST /services/KntInventoryService/GetInventory",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetInventory)))
	mux.Handle("POST /services/KntInventoryService/GetUnlockables",
		c.JWT.Middleware(http.HandlerFunc(c.handleGetUnlockables)))
	mux.Handle("POST /services/KntInventoryService/AcquireUnlockable",
		c.JWT.Middleware(http.HandlerFunc(c.handleAcquireUnlockable)))

	// KntLeaderboardService
	mux.Handle("POST /services/KntLeaderboardService/GetLeaderboards",
		c.JWT.Middleware(http.HandlerFunc(c.handleTODO)))
	mux.Handle("POST /services/KntLeaderboardService/GetFriendLeaderboards",
		c.JWT.Middleware(http.HandlerFunc(c.handleTODO)))
}

func (c *Handler) handleTODO(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}
