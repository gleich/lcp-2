package applemusic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mattglei.ch/lcp-2/internal/auth"
	"go.mattglei.ch/lcp-2/internal/cache"
	"go.mattglei.ch/lcp-2/pkg/lcp"
	"go.mattglei.ch/timber"
)

const API_ENDPOINT = "https://api.music.apple.com/"
const LOG_PREFIX = "[applemusic]"

func cacheUpdate(client *http.Client, bhCache *blurhashCache) (lcp.AppleMusicCache, error) {
	recentlyPlayed, err := fetchRecentlyPlayed(client, bhCache)
	if err != nil {
		return lcp.AppleMusicCache{}, err
	}

	playlistsIDs := []string{
		"p.AWXoXPYSLrvpJlY", // alt
		"p.AWXoZoxHLrvpJlY", // chill
		"p.V7VYVB0hZo53MQv", // old man
		"p.LV0PX3EIl0EpDLW", // jazz
		"p.gek1E8efLa68Adp", // classics
		"p.qQXLxPLtA75zg8e", // 80s
		"p.LV0PXNoCl0EpDLW", // divorced dad
		"p.QvDQE5RIVbAeokL", // PARTY
		"p.LV0PXL3Cl0EpDLW", // bops
		"p.6xZaArOsvzb5OML", // focus
		"p.O1kz7EoFVmvz704", // funk
		"p.qQXLxPpFA75zg8e", // RAHHHHHHHH
		"p.qQXLxpDuA75zg8e", // ROCK
		"p.O1kz7zbsVmvz704", // country
		"p.QvDQEN0IVbAeokL", // fall
		// "p.ZOAXAMZF4KMD6ob", // sad girl music
		// "p.QvDQEebsVbAeokL", // christmas
	}
	playlists := []lcp.AppleMusicPlaylist{}
	for _, id := range playlistsIDs {
		playlistData, err := fetchPlaylist(client, bhCache, id)
		if err != nil {
			return lcp.AppleMusicCache{}, err
		}
		playlists = append(playlists, playlistData)
	}

	return lcp.AppleMusicCache{
		RecentlyPlayed: recentlyPlayed,
		Playlists:      playlists,
	}, nil
}

func Setup(mux *http.ServeMux) {
	client := http.Client{}
	bhCache := blurhashCache{
		Entires: map[string]blurhashCacheEntry{},
	}

	data, err := cacheUpdate(&client, &bhCache)
	if err != nil {
		timber.Error(err, "initial fetch of cache data failed")
	}

	applemusicCache := cache.New("applemusic", data, err == nil)
	mux.HandleFunc("GET /applemusic", serveHTTP(applemusicCache))
	mux.HandleFunc("GET /applemusic/playlists/{id}", playlistEndpoint(applemusicCache))
	go cache.UpdatePeriodically(
		applemusicCache,
		&client,
		func(client *http.Client) (lcp.AppleMusicCache, error) {
			return cacheUpdate(client, &bhCache)
		},
		30*time.Second,
	)
	go updateAlbumArtPeriodically(&client, &bhCache, 24*time.Hour)
	timber.Done(LOG_PREFIX, "setup cache and endpoints")
}

type cacheDataResponse struct {
	PlaylistSummaries []lcp.AppleMusicPlaylistSummary `json:"playlist_summaries"`
	RecentlyPlayed    []lcp.AppleMusicSong            `json:"recently_played"`
}

func serveHTTP(c *cache.Cache[lcp.AppleMusicCache]) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAuthorized(w, r) {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		c.DataMutex.RLock()

		data := cacheDataResponse{}
		for _, p := range c.Data.Playlists {
			firstFourTracks := []lcp.AppleMusicSong{}
			for _, track := range p.Tracks {
				if len(firstFourTracks) < 4 {
					firstFourTracks = append(firstFourTracks, track)
				}
			}
			data.PlaylistSummaries = append(
				data.PlaylistSummaries,
				lcp.AppleMusicPlaylistSummary{
					Name:            p.Name,
					ID:              p.ID,
					TrackCount:      len(p.Tracks),
					FirstFourTracks: firstFourTracks,
				},
			)
		}
		data.RecentlyPlayed = c.Data.RecentlyPlayed

		err := json.NewEncoder(w).
			Encode(cache.CacheResponse[cacheDataResponse]{Data: data, Updated: c.Updated})
		c.DataMutex.RUnlock()
		if err != nil {
			err = fmt.Errorf("%w failed to write json data to request", err)
			timber.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
