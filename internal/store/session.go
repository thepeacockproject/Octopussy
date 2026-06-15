package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) GetSessionById(ctx context.Context, sessionId string) (UserSession, error) {
	var row UserSession

	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionId).
		First(&row).
		Error

	if err != nil {
		return UserSession{}, fmt.Errorf("read session by id: %w", err)
	}

	return row, nil
}

func (s *Store) GetCurrentSessionByUser(ctx context.Context, userId string) (UserSession, error) {
	var row UserSession

	err := s.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("created_at DESC").
		First(&row).
		Error

	if err != nil {
		return row, fmt.Errorf("read current user session: %w", err)
	}

	return row, nil
}

func (s *Store) CreateSession(ctx context.Context, user User, contractId string) (UserSession, error) {
	sessionId, err := uuid.NewUUID()

	if err != nil {
		panic("failed to generate session id")
	}

	row := UserSession{
		SessionId:        sessionId.String(),
		UserId:           user.Id,
		ContractId:       contractId,
		IntelCurrencyWas: user.IntelCurrency,
		RankWas:          user.Rank,
	}

	err = s.db.
		WithContext(ctx).
		Create(&row).
		Error

	if err != nil {
		return row, fmt.Errorf("create session: %w", err)
	}

	return row, nil
}
