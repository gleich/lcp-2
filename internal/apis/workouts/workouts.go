package workouts

import (
	"net/http"

	"github.com/minio/minio-go/v7"
	"go.mattglei.ch/lcp-2/internal/apis/workouts/strava"
	"go.mattglei.ch/lcp-2/pkg/lcp"
)

func fetch(
	client *http.Client,
	minioClient minio.Client,
	stravaTokens strava.Tokens,
) ([]lcp.Activity, error) {
	stravaActivities, err := strava.FetchActivities(client, minioClient, stravaTokens)
	if err != nil {
		return []lcp.Activity{}, err
	}

	return stravaActivities, nil

	// hevyWorkouts, err := hevy.FetchWorkouts(client)
	// if err != nil {
	// 	return []lcp.Activity{}, err
	// }

	// activities := []lcp.Activity{}
	// activities = append(activities, hevyWorkouts...)

	// for _, s := range stravaActivities {
	// 	conflict := false
	// 	for _, h := range hevyWorkouts {
	// 		diff := s.StartDate.Sub(h.StartDate)
	// 		if diff < 0 {
	// 			diff = -diff
	// 		}
	// 		if diff < time.Minute {
	// 			conflict = true
	// 			break
	// 		}
	// 	}
	// 	if !conflict {
	// 		activities = append(activities, s)
	// 	}
	// }

	// sort.Slice(activities, func(i, j int) bool {
	// 	return activities[i].StartDate.After(activities[j].StartDate)
	// })

	// return activities, nil
}
