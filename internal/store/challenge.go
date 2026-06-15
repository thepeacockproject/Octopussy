package store

import (
	"context"
	"fmt"
)

func (s *Store) CreateChallenge(ctx context.Context, userId string, challengeId string) (UserChallenge, error) {
	row := UserChallenge{
		UserId:      userId,
		ChallengeId: challengeId,
		Progress:    1,
	}

	err := s.db.
		WithContext(ctx).
		Create(&row).
		Error

	if err != nil {
		return row, fmt.Errorf("failed to create challengeId %s for userId %s", challengeId, userId)
	}

	return row, nil
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
