package whoopapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const workoutCollectionEndpoint = "/activity/workout"

type WorkoutQuery struct {
	Start time.Time
	End   time.Time
	Limit int
}

type WorkoutRecord struct {
	Workout Workout
	Raw     json.RawMessage
}

func (c *Client) GetWorkouts(ctx context.Context, accessToken string, query WorkoutQuery) ([]WorkoutRecord, error) {
	values := collectionQueryValues(query.Start, query.End, query.Limit)

	rawRecords, err := c.GetCollection(ctx, accessToken, workoutCollectionEndpoint, values)
	if err != nil {
		return nil, fmt.Errorf("whoop api: get workouts: %w", err)
	}

	out := make([]WorkoutRecord, 0, len(rawRecords))

	for _, raw := range rawRecords {
		var workout Workout
		if err := c.Decode(raw, &workout); err != nil {
			return nil, fmt.Errorf("whoop api: decode workout: %w", err)
		}

		workout.Raw = raw

		out = append(out, WorkoutRecord{
			Workout: workout,
			Raw:     raw,
		})
	}

	return out, nil
}
