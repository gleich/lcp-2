package strava

import (
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"pkg.mattglei.ch/lcp-2/internal/cache"
	"pkg.mattglei.ch/lcp-2/internal/secrets"
	"pkg.mattglei.ch/timber"
)

func Setup(mux *http.ServeMux) {
	stravaTokens := loadTokens()
	stravaTokens.refreshIfNeeded()
	minioClient, err := minio.New(secrets.ENV.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			secrets.ENV.MinioAccessKeyID,
			secrets.ENV.MinioSecretKey,
			"",
		),
		Secure: true,
	})
	if err != nil {
		timber.Fatal(err, "failed to create minio client")
	}
	stravaActivities, err := fetchActivities(*minioClient, stravaTokens)
	if err != nil {
		timber.Error(err, "failed to load initial data for strava cache; not updating")
	}
	stravaCache := cache.New("strava", stravaActivities, err == nil)

	mux.HandleFunc("GET /strava", stravaCache.ServeHTTP)
	mux.HandleFunc("POST /strava/event", eventRoute(stravaCache, *minioClient, stravaTokens))
	mux.HandleFunc("GET /strava/event", challengeRoute)

	timber.Done("setup strava cache")
}
