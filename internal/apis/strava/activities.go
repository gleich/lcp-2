package strava

import (
	"fmt"
	"image/png"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"pkg.mattglei.ch/lcp-2/internal/images"
	"pkg.mattglei.ch/timber"
)

type stravaActivity struct {
	Name      string    `json:"name"`
	SportType string    `json:"sport_type"`
	StartDate time.Time `json:"start_date"`
	Timezone  string    `json:"timezone"`
	Map       struct {
		SummaryPolyline string `json:"summary_polyline"`
	} `json:"map"`
	Trainer            bool    `json:"trainer"`
	Commute            bool    `json:"commute"`
	Private            bool    `json:"private"`
	AverageSpeed       float32 `json:"average_speed"`
	MaxSpeed           float32 `json:"max_speed"`
	AverageTemp        int32   `json:"average_temp,omitempty"`
	AverageCadence     float32 `json:"average_cadence,omitempty"`
	AverageWatts       float32 `json:"average_watts,omitempty"`
	DeviceWatts        bool    `json:"device_watts,omitempty"`
	AverageHeartrate   float32 `json:"average_heartrate,omitempty"`
	TotalElevationGain float32 `json:"total_elevation_gain"`
	MovingTime         uint32  `json:"moving_time"`
	SufferScore        float32 `json:"suffer_score,omitempty"`
	PrCount            uint32  `json:"pr_count"`
	Distance           float32 `json:"distance"`
	ID                 uint64  `json:"id"`
	HasHeartrate       bool    `json:"has_heartrate"`
}

type activityStream struct {
	Data         []int  `json:"data"`
	SeriesType   string `json:"series_type"`
	OriginalSize int    `json:"original_size"`
	Resolution   string `json:"resolution"`
}

type detailedStravaActivity struct {
	Calories float32 `json:"calories"`
}

type activity struct {
	Name               string    `json:"name"`
	SportType          string    `json:"sport_type"`
	StartDate          time.Time `json:"start_date"`
	Timezone           string    `json:"timezone"`
	MapBlurImage       *string   `json:"map_blur_image"`
	MapImageURL        *string   `json:"map_image_url"`
	HasMap             bool      `json:"has_map"`
	TotalElevationGain float32   `json:"total_elevation_gain"`
	MovingTime         uint32    `json:"moving_time"`
	Distance           float32   `json:"distance"`
	ID                 uint64    `json:"id"`
	AverageHeartrate   float32   `json:"average_heartrate"`
	HeartrateData      []int     `json:"heartrate_data"`
	Calories           float32   `json:"calories"`
}

func fetchActivities(
	client *http.Client,
	minioClient minio.Client,
	tokens tokens,
) ([]activity, error) {
	stravaActivities, err := sendStravaAPIRequest[[]stravaActivity](
		client,
		"api/v3/athlete/activities",
		tokens,
	)
	if err != nil {
		return nil, fmt.Errorf("%v failed to send request to Strava API to get activities", err)
	}

	var activities []activity
	for _, stravaActivity := range stravaActivities {
		if len(activities) >= 5 {
			break
		}
		if stravaActivity.Private || !stravaActivity.HasHeartrate {
			continue
		}

		details, err := fetchActivityDetails(client, stravaActivity.ID, tokens)
		if err != nil {
			timber.Error(err, "failed to fetch activity details")
			continue
		}

		hrStream, err := fetchHeartrate(client, stravaActivity.ID, tokens)
		if err != nil {
			return nil, fmt.Errorf("%v failed to fetch HR data", err)
		}

		a := activity{
			Name:               stravaActivity.Name,
			SportType:          stravaActivity.SportType,
			StartDate:          stravaActivity.StartDate,
			Timezone:           stravaActivity.Timezone,
			TotalElevationGain: stravaActivity.TotalElevationGain,
			MovingTime:         stravaActivity.MovingTime,
			Distance:           stravaActivity.Distance,
			ID:                 stravaActivity.ID,
			AverageHeartrate:   stravaActivity.AverageHeartrate,
			HasMap:             stravaActivity.Map.SummaryPolyline != "",
			HeartrateData:      hrStream,
			Calories:           details.Calories,
		}

		if a.HasMap {
			mapData := fetchMap(stravaActivity.Map.SummaryPolyline)
			uploadMap(minioClient, stravaActivity.ID, mapData)
			mapBlurData, err := images.BlurImage(mapData, png.Decode)
			if err != nil {
				timber.Error(err, "failed to create blur image")
				continue
			}
			mapBlurURI := images.BlurDataURI(mapBlurData)
			a.MapBlurImage = &mapBlurURI
			imgurl := fmt.Sprintf(
				"https://minio-api.dev.mattglei.ch/mapbox-maps/%d.png",
				a.ID,
			)
			a.MapImageURL = &imgurl
		}
		activities = append(activities, a)
	}
	removeOldMaps(minioClient, activities)

	return activities, nil
}

func fetchHeartrate(client *http.Client, id uint64, tokens tokens) ([]int, error) {
	params := url.Values{
		"key_by_type": {"true"},
		"keys":        {"heartrate"},
		"resolution":  {"low"},
	}
	stream, err := sendStravaAPIRequest[struct{ Heartrate activityStream }](
		client,
		fmt.Sprintf("api/v3/activities/%d/streams?%s", id, params.Encode()),
		tokens,
	)
	if err != nil {
		return []int{}, fmt.Errorf(
			"%v failed to send request for HR data from activity with ID of %d",
			err,
			id,
		)
	}

	return stream.Heartrate.Data, nil
}

func fetchActivityDetails(
	client *http.Client,
	id uint64,
	tokens tokens,
) (detailedStravaActivity, error) {
	details, err := sendStravaAPIRequest[detailedStravaActivity](
		client,
		fmt.Sprintf("api/v3/activities/%d", id),
		tokens,
	)
	if err != nil {
		return detailedStravaActivity{}, fmt.Errorf(
			"%v failed to request detailed activity data for %d",
			err,
			id,
		)
	}

	return details, nil
}
