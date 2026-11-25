package persistence

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SetEntitlements atomically replaces the user's stored entitlements.
func (s *Store) SetEntitlements(ctx context.Context, userId string, entitlements []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userId).Delete(&UserEntitlement{}).Error; err != nil {
			return fmt.Errorf("clear entitlements: %w", err)
		}
		if len(entitlements) == 0 {
			return nil
		}

		rows := make([]UserEntitlement, 0, len(entitlements))
		for _, e := range entitlements {
			rows = append(rows, UserEntitlement{UserId: userId, Entitlement: e})
		}
		if err := tx.Omit("User").Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return fmt.Errorf("store entitlements: %w", err)
		}
		return nil
	})
}

// GetEntitlements returns the user's stored entitlements, sorted.
func (s *Store) GetEntitlements(ctx context.Context, userId string) ([]string, error) {
	var out []string
	err := s.db.WithContext(ctx).
		Model(&UserEntitlement{}).
		Where("user_id = ?", userId).
		Order("entitlement").
		Pluck("entitlement", &out).
		Error
	if err != nil {
		return nil, fmt.Errorf("get entitlements: %w", err)
	}
	return out, nil
}
