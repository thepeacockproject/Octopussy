package store

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"
)

// CustomProfileData accessors back the CustomProfileDataService. The game stores
// arbitrary client-side state (menu flags, "new item seen" lists, checkpoint
// blobs) as opaque string values — Octopussy persists and returns them verbatim.

// ReadCustomProfileData returns every stored key/value pair for a profile. A
// profile with nothing stored yet returns an empty (non-nil) map; callers can
// seed defaults via SeedCustomProfileDefaults.
func (s *Store) ReadCustomProfileData(ctx context.Context, profileID string) (map[string]string, error) {
	var rows []CustomProfileData
	if err := s.db.WithContext(ctx).
		Where("profile_id = ?", profileID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read custom profile data: %w", err)
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Data
	}
	return out, nil
}

// WriteCustomProfileData upserts a single key/value pair for a profile. This is
// the persistence behind CustomProfileDataService/Write.
func (s *Store) WriteCustomProfileData(ctx context.Context, profileID, key, data string) error {
	rec := CustomProfileData{ProfileID: profileID, Key: key, Data: data}
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "profile_id"}, {Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"data"}),
		}).
		Create(&rec).Error
	if err != nil {
		return fmt.Errorf("write custom profile data: %w", err)
	}
	return nil
}

// CountCustomProfileData reports how many key/value pairs a profile has stored.
// It is used to decide whether a profile is brand new and needs default seeding.
func (s *Store) CountCustomProfileData(ctx context.Context, profileID string) (int, error) {
	var n int64
	if err := s.db.WithContext(ctx).
		Model(&CustomProfileData{}).
		Where("profile_id = ?", profileID).
		Count(&n).Error; err != nil {
		return 0, fmt.Errorf("count custom profile data: %w", err)
	}
	return int(n), nil
}

// SeedCustomProfileDefaults writes the given defaults for a profile, but only
// for keys it doesn't already have. Existing values are left untouched (ON
// CONFLICT DO NOTHING), so this is safe to call on every ReadAll.
func (s *Store) SeedCustomProfileDefaults(ctx context.Context, profileID string, defaults map[string]string) error {
	if len(defaults) == 0 {
		return nil
	}

	recs := make([]CustomProfileData, 0, len(defaults))
	for key, data := range defaults {
		recs = append(recs, CustomProfileData{ProfileID: profileID, Key: key, Data: data})
	}

	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&recs).Error
	if err != nil {
		return fmt.Errorf("seed custom profile defaults: %w", err)
	}
	return nil
}
