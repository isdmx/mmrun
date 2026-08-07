package client

import (
	"sort"

	"github.com/mattermost/mattermost/server/public/model"
)

// SortPosts returns the posts of a PostList sorted oldest-first. It is
// resilient to a nil list and to Order/Posts inconsistencies.
func SortPosts(pl *model.PostList) []*model.Post {
	if pl == nil {
		return nil
	}
	posts := make([]*model.Post, 0, len(pl.Posts))
	for _, p := range pl.Posts {
		if p != nil {
			posts = append(posts, p)
		}
	}
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].CreateAt < posts[j].CreateAt })
	return posts
}
