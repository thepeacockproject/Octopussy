package store

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migrations returns the ordered list of schema migrations. Each has a stable,
// sortable ID and an up (Migrate) / down (Rollback) pair, applied exactly once
// and tracked in the migrations table. Append new entries; never edit or
// reorder existing ones once they've shipped.
func migrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		{
			// Initial schema: the CustomProfileDataService key/value store.
			ID: "0001_custom_profile_data",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&CustomProfileData{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&CustomProfileData{})
			},
		},
		{
			// User accounts and their external platform identities.
			ID: "0002_users_and_identities",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&User{}, &UserIdentity{})
			},
			Rollback: func(tx *gorm.DB) error {
				// Drop the child (identities) before the parent (users).
				return tx.Migrator().DropTable(&UserIdentity{}, &User{})
			},
		},
		{
			// Rename custom_profile_data.profile_id to user_id and add a foreign
			// key into users.id, tying stored profile state to a user account.
			ID: "0003_custom_profile_data_user_fk",
			Migrate: func(tx *gorm.DB) error {
				if tx.Migrator().HasColumn(&CustomProfileData{}, "profile_id") {
					if err := tx.Migrator().RenameColumn(&CustomProfileData{}, "profile_id", "user_id"); err != nil {
						return err
					}
				}
				// AutoMigrate adds the user_id -> users.id foreign key constraint.
				return tx.AutoMigrate(&CustomProfileData{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().RenameColumn(&CustomProfileData{}, "user_id", "profile_id")
			},
		},
		{
			// Add intel_currency (default 0) and rank (default 1) to users, and
			// create the user_sessions table tracking a user's intel currency and
			// rank as they were at the start of each play session.
			ID: "0004_user_sessions_and_currency",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&User{}, &UserSession{})
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&UserSession{}); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&User{}, "intel_currency"); err != nil {
					return err
				}
				return tx.Migrator().DropColumn(&User{}, "rank")
			},
		},
		{
			// Add contract_id to user_sessions
			ID: "0005_session_contract_id",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&UserSession{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&UserSession{}, "contract_id")
			},
		},
		{
			// Add created_at to user_sessions so the most recent session for a
			// user can be ordered and found.
			ID: "0006_session_created_at",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&UserSession{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&UserSession{}, "created_at")
			},
		},
		{
			// Add xp_was to user_sessions
			ID: "0007_user_session_xp_was",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&UserSession{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&UserSession{}, "xp_was")
			},
		},
		{
			// Add challenges and unlockables
			ID: "0008_challenges_unlockables",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&UserChallenge{}, &UserUnlockable{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&UserChallenge{}, &UserUnlockable{})
			},
		},
	}
}
