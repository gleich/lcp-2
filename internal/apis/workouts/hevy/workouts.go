package hevy

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.mattglei.ch/lcp-2/internal/secrets"
	"go.mattglei.ch/lcp-2/pkg/lcp"
)

type workoutsResponse struct {
	Workouts []struct {
		ID        string             `json:"id"`
		Title     string             `json:"title"`
		StartTime time.Time          `json:"start_time"`
		EndTime   time.Time          `json:"end_time"`
		CreatedAt time.Time          `json:"created_at"`
		Exercises []lcp.HevyExercise `json:"exercises"`
	} `json:"workouts"`
}

func FetchWorkouts(client *http.Client) ([]lcp.Activity, error) {
	params := url.Values{"api-key": {secrets.ENV.HevyAccessToken}}
	workouts, err := sendHevyAPIRequest[workoutsResponse](
		client,
		fmt.Sprintf("/v1/workouts?%s", params.Encode()),
	)
	if err != nil {
		return []lcp.Activity{}, fmt.Errorf("%w ", err)
	}

	var activities []lcp.Activity
	for _, workout := range workouts.Workouts {
		volume := 0.0
		for _, exercise := range workout.Exercises {
			for _, set := range exercise.Sets {
				volume += set.WeightKg
			}
		}
		activities = append(activities, lcp.Activity{
			Platform:      "hevy",
			Name:          workout.Title,
			StartDate:     workout.StartTime.UTC(),
			MovingTime:    uint32(workout.EndTime.Sub(workout.StartTime).Seconds()),
			SportType:     "WeightTraining",
			HasMap:        false,
			ID:            workout.ID,
			HasHeartrate:  false,
			HevyExercises: workout.Exercises,
			HevyVolumeKG:  volume,
		})
	}

	return activities, nil
}
