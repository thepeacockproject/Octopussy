package persistence

import (
	"context"
	"fmt"

	"gorm.io/gorm/clause"
)

// CreateChallenge records challengeId as completed for userId. It is idempotent:
// if the row already exists nothing is written and created is false, so callers
// can avoid granting rewards twice when the game retries a completion.
func (s *Store) CreateChallenge(ctx context.Context, userId string, challengeId string) (row UserChallenge, created bool, err error) {
	row = UserChallenge{
		UserId:      userId,
		ChallengeId: challengeId,
		Progress:    1,
	}

	res := s.db.
		WithContext(ctx).
		Omit("User").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row)

	if res.Error != nil {
		return row, false, fmt.Errorf("failed to create challengeId %s for userId %s: %w", challengeId, userId, res.Error)
	}

	return row, res.RowsAffected > 0, nil
}

func (s *Store) GetChallenges(ctx context.Context, userId string) ([]UserChallenge, error) {
	var rows []UserChallenge

	err := s.db.
		WithContext(ctx).
		Where("user_id = ?", userId).
		Find(&rows).
		Error

	if err != nil {
		return rows, fmt.Errorf("failed to retrieve any challenges for userId %s", userId)
	}

	return rows, nil
}

func (s *Store) GetChallengesByIds(ctx context.Context, userId string, challengeIds []string) ([]UserChallenge, error) {
	var rows []UserChallenge

	err := s.db.
		WithContext(ctx).
		Where("user_id = ? AND challenge_id IN (?)", userId, challengeIds).
		Find(&rows).
		Error

	if err != nil {
		return rows, fmt.Errorf("failed to retrieve any challengeIds for userId %s", userId)
	}

	return rows, nil
}
