package store

// GORM models. Schema changes to these structs must be paired with a migration
// in migrations.go - AutoMigrate is only ever invoked from inside a versioned
// migration, never implicitly at startup, so the schema stays reproducible.

// CustomProfileData is one entry in a profile's CustomProfileDataService bag: an
// opaque string value stored under a key, scoped to a profile. The primary key
// is the (profile_id, key) pair.
type CustomProfileData struct {
	ProfileID string `gorm:"primaryKey;column:profile_id"`
	Key       string `gorm:"primaryKey;column:key"`
	Data      string `gorm:"column:data;not null"`
}

// TableName pins the table name so it doesn't depend on GORM's pluralization.
func (CustomProfileData) TableName() string { return "custom_profile_data" }
