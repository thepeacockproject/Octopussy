package persistence

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SetEntitlements atomically replaces the identity's stored entitlements.
func (s *Store) SetEntitlements(ctx context.Context, platform, platformId string, entitlements []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var identity UserIdentity
		err := tx.Select("id").
			Where("platform = ? AND platform_id = ?", platform, platformId).
			Take(&identity).
			Error
		if err != nil {
			return fmt.Errorf("find identity %s/%s: %w", platform, platformId, err)
		}

		if err := tx.Where("identity_id = ?", identity.Id).Delete(&IdentityEntitlement{}).Error; err != nil {
			return fmt.Errorf("clear entitlements: %w", err)
		}
		if len(entitlements) == 0 {
			return nil
		}

		rows := make([]IdentityEntitlement, 0, len(entitlements))
		for _, e := range entitlements {
			rows = append(rows, IdentityEntitlement{IdentityId: identity.Id, Entitlement: e})
		}
		if err := tx.Omit("Identity").Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return fmt.Errorf("store entitlements: %w", err)
		}
		return nil
	})
}

// GetEntitlements returns the identity's stored entitlements, sorted.
func (s *Store) GetEntitlements(ctx context.Context, platform, platformId string) ([]string, error) {
	identity := s.db.
		Model(&UserIdentity{}).
		Select("id").
		Where("platform = ? AND platform_id = ?", platform, platformId)

	out := []string{}
	err := s.db.WithContext(ctx).
		Model(&IdentityEntitlement{}).
		Where("identity_id IN (?)", identity).
		Order("entitlement").
		Pluck("entitlement", &out).
		Error
	if err != nil {
		return nil, fmt.Errorf("get entitlements: %w", err)
	}
	return out, nil
}
