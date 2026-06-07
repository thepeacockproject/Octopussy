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
	}
}
