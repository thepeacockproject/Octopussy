package store

import (
	"context"
	"fmt"
)

func (s *Store) CreateUnlockable(ctx context.Context, userId string, unlockableId string) (UserUnlockable, error) {
	row := UserUnlockable{
		UserID:       userId,
		UnlockableID: unlockableId,
	}

	if s.db.WithContext(ctx).Create(&row).Error != nil {
		return row, fmt.Errorf("failed to create unlockableId %s for userId %s", unlockableId, userId)
	}

	return row, nil
}

func (s *Store) GetUnlockables(ctx context.Context, userId string) ([]UserUnlockable, error) {
	var rows []UserUnlockable
	err := s.db.
		WithContext(ctx).
		Where("user_id = ?", userId).
		Find(&rows)

	if err != nil {
		return rows, fmt.Errorf("failed to retrieve any unlockables for userId %s", userId)
	}

	return rows, nil
}

func (s *Store) GetUnlockableById(ctx context.Context, userId string, unlockableId string) (UserUnlockable, error) {
	var row UserUnlockable
	err := s.db.
		WithContext(ctx).
		Where("user_id = ? AND unlockable_id", userId, unlockableId).
		First(&row)

	if err != nil {
		return row, fmt.Errorf("failed to retrieve unlockable %s for userId %s", unlockableId, userId)
	}

	return row, nil
}
