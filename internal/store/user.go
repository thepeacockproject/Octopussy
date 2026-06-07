package store

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// User accessors back player accounts and their external platform identities.

// SetUserXP sets a user's accumulated XP to an absolute value.
func (s *Store) SetUserXP(ctx context.Context, userID string, xp int64) error {
	err := s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Update("xp_earned", xp).Error
	if err != nil {
		return fmt.Errorf("set user xp: %w", err)
	}
	return nil
}

// UserXP returns a user's accumulated XP. An unknown user reports zero XP rather
// than an error, matching a brand-new profile with no progression yet.
func (s *Store) UserXP(ctx context.Context, userID string) (int64, error) {
	var user User
	err := s.db.WithContext(ctx).Select("xp_earned").First(&user, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read user xp: %w", err)
	}
	return user.XPEarned, nil
}

// User loads a user by ID.
func (s *Store) User(ctx context.Context, userID string) (*User, error) {
	var user User
	return &user, s.db.WithContext(ctx).First(&user, "id = ?", userID).Error
}

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

// ResolvedProfile is a user together with its external platform identities,
// keyed platform -> platform id. It backs SharedProfileService/ResolveProfiles.
type ResolvedProfile struct {
	ID             string
	Gamertag       string
	LinkedAccounts map[string]string
}

// ResolveProfiles looks up the given user ids and returns each as a
// ResolvedProfile with its gamertag and linked platform accounts. Results follow
// the order of the requested ids; ids with no matching user are skipped, so the
// returned slice may be shorter than the input.
func (s *Store) ResolveProfiles(ctx context.Context, ids []string) ([]ResolvedProfile, error) {
	if len(ids) == 0 {
		return []ResolvedProfile{}, nil
	}

	var users []User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("resolve profiles: load users: %w", err)
	}

	var identities []UserIdentity
	if err := s.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&identities).Error; err != nil {
		return nil, fmt.Errorf("resolve profiles: load identities: %w", err)
	}

	// Group identities by user into platform -> platform_id maps.
	linked := make(map[string]map[string]string, len(users))
	for _, id := range identities {
		accounts := linked[id.UserID]
		if accounts == nil {
			accounts = make(map[string]string)
			linked[id.UserID] = accounts
		}
		accounts[id.Platform] = id.PlatformID
	}

	byID := make(map[string]User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}

	out := make([]ResolvedProfile, 0, len(ids))
	for _, id := range ids {
		u, ok := byID[id]
		if !ok {
			continue
		}
		accounts := linked[id]
		if accounts == nil {
			accounts = map[string]string{}
		}
		out = append(out, ResolvedProfile{
			ID:             u.ID,
			Gamertag:       u.Gamertag,
			LinkedAccounts: accounts,
		})
	}
	return out, nil
}
