package applemusic

import (
	"fmt"
	"net/http"

	"pkg.mattglei.ch/lcp-2/pkg/lcp"
)

type recentlyPlayedResponse struct {
	Data []songResponse `json:"data"`
}

func fetchRecentlyPlayed(client *http.Client) ([]lcp.AppleMusicSong, error) {
	response, err := sendAppleMusicAPIRequest[recentlyPlayedResponse](
		client,
		"/v1/me/recent/played/tracks",
	)
	if err != nil {
		return []lcp.AppleMusicSong{}, err
	}

	var songs []lcp.AppleMusicSong
	for _, s := range response.Data {
		so, err := songFromSongResponse(s)
		if err != nil {
			return []lcp.AppleMusicSong{}, fmt.Errorf(
				"%v failed to parse song from song response",
				err,
			)
		}
		songs = append(songs, so)
	}

	// filter out duplicate songs
	seen := make(map[string]bool)
	uniqueSongs := []lcp.AppleMusicSong{}
	for _, song := range songs {
		if !seen[song.ID] {
			seen[song.ID] = true
			uniqueSongs = append(uniqueSongs, song)
		}
	}

	return uniqueSongs[:10], nil
}
