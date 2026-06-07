package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) GetSessionByID(ctx context.Context, sessionID string) (UserSession, error) {
	var row UserSession
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&row).Error; err != nil {
		return UserSession{}, fmt.Errorf("read session by id: %w", err)
	}
	return row, nil
}

func (s *Store) GetCurrentSessionByUser(ctx context.Context, userID string) (UserSession, error) {
	var row UserSession
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&row).Error; err != nil {
		return row, fmt.Errorf("read current user session: %w", err)
	}
	return row, nil
}

func (s *Store) CreateSession(ctx context.Context, user User, contractID string) (UserSession, error) {
	sessionID, err := uuid.NewUUID()

	if err != nil {
		panic("failed to generate session id")
	}

	row := UserSession{
		SessionID:        sessionID.String(),
		UserID:           user.ID,
		ContractID:       contractID,
		IntelCurrencyWas: user.IntelCurrency,
		RankWas:          user.Rank,
	}

	if s.db.WithContext(ctx).
		Create(&row).Error != nil {
		return row, fmt.Errorf("create session: %w", err)
	}

	return row, nil
}
