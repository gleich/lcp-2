package github

import (
	"context"
	"fmt"
	"time"

	"github.com/gleich/lumber/v2"
	"github.com/shurcooL/githubv4"
)

type pinnedItemsQuery struct {
	Viewer struct {
		PinnedItems struct {
			Nodes []struct {
				Repository struct {
					Name  githubv4.String
					Owner struct {
						Login githubv4.String
					}
					PrimaryLanguage struct {
						Name  githubv4.String
						Color githubv4.String
					}
					Description    githubv4.String
					UpdatedAt      githubv4.DateTime
					StargazerCount githubv4.Int
					IsPrivate      githubv4.Boolean
					ID             githubv4.ID
					URL            githubv4.URI
				} `graphql:"... on Repository"`
			}
		} `graphql:"pinnedItems(first: 6, types: REPOSITORY)"`
	}
}

type repository struct {
	Name          string    `json:"name"`
	Owner         string    `json:"owner"`
	Language      string    `json:"language"`
	LanguageColor string    `json:"language_color"`
	Description   string    `json:"description"`
	UpdatedAt     time.Time `json:"updated_at"`
	Stargazers    int32     `json:"stargazers"`
	ID            string    `json:"id"`
	URL           string    `json:"url"`
}

func FetchPinnedRepos(client *githubv4.Client) []repository {
	var query pinnedItemsQuery
	err := client.Query(context.Background(), &query, nil)
	if err != nil {
		lumber.Error(err, "querying github's graphql API failed")
		return nil
	}

	var repositories []repository
	for _, node := range query.Viewer.PinnedItems.Nodes {
		repositories = append(repositories, repository{
			Name:          string(node.Repository.Name),
			Owner:         string(node.Repository.Owner.Login),
			Language:      string(node.Repository.PrimaryLanguage.Name),
			LanguageColor: string(node.Repository.PrimaryLanguage.Color),
			Description:   string(node.Repository.Description),
			UpdatedAt:     node.Repository.UpdatedAt.Time,
			Stargazers:    int32(node.Repository.StargazerCount),
			ID:            fmt.Sprint(node.Repository.ID),
			URL:           fmt.Sprint(node.Repository.URL.URL),
		})
	}
	return repositories
}
