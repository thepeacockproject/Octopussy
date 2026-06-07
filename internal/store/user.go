package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// User accessors back player accounts and their external platform identities.

// EnsureUser makes sure a user and the given platform identity both exist,
// creating them on first login and leaving any existing rows untouched. It is
// called during OAuth so every authenticated account is backed by a persisted
// user. userID is the canonical profile id (stored as users.id); gamertag is the
// initial display name used only when the user is first created; platform and
// platformID are the external identity (e.g. "steam" + the steam user id).
func (s *Store) EnsureUser(ctx context.Context, userID, gamertag, platform, platformID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := User{ID: userID, Gamertag: gamertag}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&user).Error; err != nil {
			return fmt.Errorf("ensure user: %w", err)
		}

		// Omit the User association so writing an identity never touches the
		// users table through it; the user is created explicitly above.
		identity := UserIdentity{UserID: userID, Platform: platform, PlatformID: platformID}
		if err := tx.Omit("User").Clauses(clause.OnConflict{DoNothing: true}).Create(&identity).Error; err != nil {
			return fmt.Errorf("ensure user identity: %w", err)
		}
		return nil
	})
}
