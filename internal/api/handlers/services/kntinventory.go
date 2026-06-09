package services

import (
	"net/http"
	"strings"

	"github.com/samber/lo"
	"github.com/thepeacockproject/Octopussy/internal/api/shared"
	"github.com/thepeacockproject/Octopussy/internal/content"
	"github.com/thepeacockproject/Octopussy/internal/store"
)

type getInventoryRequest struct {
	ProfileID string `json:"profileId"`
}

type getInventoryResponse struct {
	UnlockableItems []unlockableItem `json:"UnlockableItems"`
	Wallet          wallet           `json:"Wallet"`
}

type unlockableItem struct {
	Id     string `json:"Id"`
	Amount int    `json:"Amount"`
	Type   string `json:"Type"`
	IsNew  bool   `json:"IsNew"`
}

type wallet struct {
	Currency currency `json:"Currency"`
}

type currency struct {
	Intel int `json:"Intel"`
}

type getUnlockablesRequest struct {
	ProfileID string `json:"profileId"`
}

//type getUnlockablesResponse = []content.UnlockableWithState

type acquireUnlockableRequest struct {
	ProfileID     string `json:"profileId"`
	UnlockableId  string `json:"unlockableId"`
	PriceCurrency string `json:"priceCurrency"`
	PriceAmount   int    `json:"priceAmount"`
}

func (c *Handler) handleGetInventory(w http.ResponseWriter, r *http.Request) {
	_, ok := shared.ParseBody[getInventoryRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	profileId := profileIDFor(r)

	unlockables, err := c.Store.GetUnlockables(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockableItems := lo.Map(unlockables, func(item store.UserUnlockable, index int) unlockableItem {
		return unlockableItem{
			Id:     item.UnlockableID,
			Amount: 1,
			Type:   "",
			IsNew:  false,
		}
	})

	user, err := c.Store.User(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.WriteJSON(w, http.StatusOK, getInventoryResponse{
		UnlockableItems: unlockableItems,
		Wallet: wallet{
			Currency: currency{
				Intel: user.IntelCurrency,
			},
		},
	})
}

func (c *Handler) handleGetUnlockables(w http.ResponseWriter, r *http.Request) {
	_, ok := shared.ParseBody[getUnlockablesRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	profileId := profileIDFor(r)

	unlockables, err := c.Store.GetUnlockables(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := c.Store.User(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	profileData, err := c.Store.ReadCustomProfileData(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	profileDataCheckpoints, ok := profileData["CustomProfileCheckpoints"]
	if !ok {
		// NOTE: We're not actually going to parse the JSON inside
		profileDataCheckpoints = ""
	}

	result := lo.Map(c.Content.Unlockables(), func(u content.Unlockable, index int) content.UnlockableWithState {
		return mapToUnlockableWithState(u, user, unlockables, profileDataCheckpoints)
	})

	c.WriteJSON(w, http.StatusOK, result)
}

// TODO: Clean this up
func (c *Handler) handleAcquireUnlockable(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[acquireUnlockableRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Only support Intel for now
	if body.PriceCurrency != "Intel" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	profileId := profileIDFor(r)

	user, err := c.Store.User(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	profileData, err := c.Store.ReadCustomProfileData(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	profileDataCheckpoints, ok := profileData["CustomProfileCheckpoints"]
	if !ok {
		// NOTE: We're not actually going to parse the JSON inside
		profileDataCheckpoints = ""
	}

	userUnlockable, err := c.Store.GetUnlockableById(ctx, profileId, body.UnlockableId)
	if err == nil {
		// Player already owns this item
		c.WriteJSON(w, http.StatusOK, false)
		return
	}

	staticUnlockable, ok := c.Content.Unlockable(body.UnlockableId)
	if !ok || staticUnlockable.PriceCurrency != "Intel" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockableWithState := mapToUnlockableWithState(
		staticUnlockable,
		user,
		[]store.UserUnlockable{userUnlockable},
		profileDataCheckpoints,
	)

	if !unlockableWithState.CanBuy {
		// Player can't buy this item
		c.WriteJSON(w, http.StatusOK, false)
		return
	}

	_, err = c.Store.CreateUnlockable(ctx, profileId, body.UnlockableId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.WriteJSON(w, http.StatusOK, true)
}

func mapToUnlockableWithState(unlockable content.Unlockable, user *store.User, unlockables []store.UserUnlockable, profileDataCheckpoints string) content.UnlockableWithState {
	unlockableWithState := content.UnlockableWithState{
		Unlockable: unlockable,
		LevelOk:    user.Rank >= unlockable.RequiredLevel,
		PriceOk:    unlockable.PriceCurrency == "Intel" && user.IntelCurrency >= unlockable.PriceValue,
	}

	if lo.ContainsBy(unlockables, func(item store.UserUnlockable) bool {
		return item.UnlockableID == unlockableWithState.UnlockableState.Id
	}) {
		unlockableWithState.UnlockableState.State = content.UnlockState_Unlocked
	} else {
		unlockableWithState.UnlockableState.State = content.UnlockState_Locked
	}

	checkpointOk := unlockableWithState.RequiredCheckpoint == "" ||
		strings.Contains(profileDataCheckpoints, unlockableWithState.RequiredCheckpoint)

	// A player can only buy an item if it's available for sale,
	// have the required level, have the required currency, have the required checkpoint
	// and don't already own the item.
	unlockableWithState.CanBuy = unlockableWithState.IntelShop &&
		unlockableWithState.LevelOk &&
		unlockableWithState.PriceOk &&
		checkpointOk &&
		unlockableWithState.UnlockableState.State == content.UnlockState_Locked

	return unlockableWithState
}
