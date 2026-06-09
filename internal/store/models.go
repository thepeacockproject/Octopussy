package store

import "time"

// GORM models. Schema changes to these structs must be paired with a migration
// in migrations.go - AutoMigrate is only ever invoked from inside a versioned
// migration, never implicitly at startup, so the schema stays reproducible.

type UserUnlockable struct {
	UserID       string `gorm:"primaryKey;column:user_id"`
	UnlockableID string `gorm:"primaryKey;column:unlockable_id"`
	User         User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

type UserChallenge struct {
	UserID      string `gorm:"primaryKey;column:user_id"`
	ChallengeID string `gorm:"primaryKey;column:challenge_id"`
	Progress    int    `gorm:"column:progress_current"`
	User        User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// CustomProfileData is one entry in a profile's CustomProfileDataService bag: an
// opaque string value stored under a key, scoped to a user. The primary key is
// the (user_id, key) pair, and user_id is a foreign key into users.id.
type CustomProfileData struct {
	UserID string `gorm:"primaryKey;column:user_id"`
	Key    string `gorm:"primaryKey;column:key"`
	Data   string `gorm:"column:data;not null"`
	User   User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName pins the table name because "data" should not be pluralized.
func (CustomProfileData) TableName() string { return "custom_profile_data" }

// User is a single player account. ID is the canonical profile id (the stable
// UUID derived from the external identity at login, also used as the JWT
// subject). A user can sign in from several platforms; those are tracked as
// UserIdentity rows. XPEarned backs player progression and starts at zero.
type User struct {
	ID       string `gorm:"primaryKey;column:id"`
	Gamertag string `gorm:"column:gamertag;not null"`
	XPEarned int64  `gorm:"column:xp_earned;not null;default:0"`
	// IntelCurrency is the player's spendable in-game currency (starts at 0)
	IntelCurrency int `gorm:"column:intel_currency;not null;default:0"`
	// Rank is the player's current rank (starts at 1).
	Rank int `gorm:"column:rank;not null;default:1"`
}

// UserIdentity links an external platform login (e.g. Steam, Epic, Apple) to a
// User. The (platform, platform_id) pair is unique so the same external account
// always resolves to one user, and UserID is a foreign key into users.id.
type UserIdentity struct {
	ID         uint   `gorm:"primaryKey;column:id"`
	UserID     string `gorm:"column:user_id;not null;index"`
	Platform   string `gorm:"column:platform;not null;uniqueIndex:idx_user_identities_platform_id"`
	PlatformID string `gorm:"column:platform_id;not null;uniqueIndex:idx_user_identities_platform_id"`
	User       User   `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

// UserSession is one play session for a user.
type UserSession struct {
	// SessionID is the session's UUID (the primary key)
	SessionID string `gorm:"primaryKey;column:session_id"`
	// UserID is a foreign key into users.id
	UserID string `gorm:"column:user_id;not null;index"`
	// IntelCurrencyWas value of users.intel_currency when session started.
	IntelCurrencyWas int `gorm:"column:intel_currency_was;not null"`
	// RankWas value of users.rank when session started.
	RankWas int `gorm:"column:rank_was;not null"`
	// ContractID the contract's ID.
	ContractID string `gorm:"column:contract_id"`
	// CreatedAt is set by GORM when the session is created; it orders sessions so
	// the most recent can be found.
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	// XpWas value of users.xp_earned when session started.
	XpWas int  `gorm:"column:xp_was;not null"`
	User  User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}
