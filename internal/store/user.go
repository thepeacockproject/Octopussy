package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Profile struct {
	Id             string
	Gamertag       string
	LinkedAccounts map[string]string
}

// EnsureUser makes sure a user and the given platform identity both exist,
// creating them on first login and leaving any existing rows untouched. It is
// called during OAuth so every authenticated account is backed by a persisted
// user. userId is the canonical profile id (stored as users.id); gamertag is the
// initial display name used only when the user is first created; platform and
// platformId are the external identity (e.g. "steam" + the steam user id).
func (s *Store) EnsureUser(ctx context.Context, userId, gamertag, platform, platformId string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := User{Id: userId, Gamertag: gamertag}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&user).Error; err != nil {
			return fmt.Errorf("ensure user: %w", err)
		}

		// Omit the User association so writing an identity never touches the
		// users table through it; the user is created explicitly above.
		identity := UserIdentity{UserId: userId, Platform: platform, PlatformId: platformId}
		if err := tx.Omit("User").Clauses(clause.OnConflict{DoNothing: true}).Create(&identity).Error; err != nil {
			return fmt.Errorf("ensure user identity: %w", err)
		}

		return nil
	})
}

func (s *Store) GetUser(ctx context.Context, userId string) (*User, error) {
	var user User

	err := s.db.
		WithContext(ctx).
		First(&user, "id = ?", userId).
		Error

	return &user, err
}

func (s *Store) UpdateUser(ctx context.Context, user *User) error {
	err := s.db.
		WithContext(ctx).
		Model(&User{}).
		Where("id = ?", user.Id).
		Updates(User{
			Gamertag:      user.Gamertag,
			XPEarned:      user.XPEarned,
			IntelCurrency: user.IntelCurrency,
			Rank:          user.Rank,
		}).
		Error

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// GetProfilesByIds looks up the given user ids and returns each as a
// Profile with its gamertag and linked platform accounts. Results follow
// the order of the requested ids; ids with no matching user are skipped, so the
// returned slice may be shorter than the input.
func (s *Store) GetProfilesByIds(ctx context.Context, ids []string) ([]Profile, error) {
	if len(ids) == 0 {
		return []Profile{}, nil
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
		accounts := linked[id.UserId]
		if accounts == nil {
			accounts = make(map[string]string)
			linked[id.UserId] = accounts
		}
		accounts[id.Platform] = id.PlatformId
	}

	byId := make(map[string]User, len(users))
	for _, u := range users {
		byId[u.Id] = u
	}

	out := make([]Profile, 0, len(ids))
	for _, id := range ids {
		u, ok := byId[id]
		if !ok {
			continue
		}
		accounts := linked[id]
		if accounts == nil {
			accounts = map[string]string{}
		}
		out = append(out, Profile{
			Id:             u.Id,
			Gamertag:       u.Gamertag,
			LinkedAccounts: accounts,
		})
	}

	return out, nil
}
