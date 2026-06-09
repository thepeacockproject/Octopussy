package store

import (
	"context"
	"fmt"
)

func (s *Store) CreateChallenge(ctx context.Context, userId string, challengeId string) (UserChallenge, error) {
	row := UserChallenge{
		UserID:      userId,
		ChallengeID: challengeId,
		Progress:    1,
	}

	if s.db.WithContext(ctx).Create(&row).Error != nil {
		return row, fmt.Errorf("failed to create challengeId %s for userId %s", challengeId, userId)
	}

	return row, nil
}

func (s *Store) GetChallengesById(ctx context.Context, userId string, challengeIds []string) ([]UserChallenge, error) {
	var rows []UserChallenge
	err := s.db.
		WithContext(ctx).
		Where("user_id = ? AND challenge_id IN (?)", userId, challengeIds).
		Find(&rows)

	if err != nil {
		return rows, fmt.Errorf("failed to retrieve any challengeIds for userId %s", userId)
	}

	return rows, nil
}
