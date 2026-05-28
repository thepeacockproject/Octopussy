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
	}
}
