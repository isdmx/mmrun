package client

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestSortPosts_OldestFirst(t *testing.T) {
	pl := &model.PostList{
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", CreateAt: 3000},
			"p2": {Id: "p2", CreateAt: 1000},
			"p3": {Id: "p3", CreateAt: 2000},
		},
	}
	posts := SortPosts(pl)
	if len(posts) != 3 {
		t.Fatalf("got %d posts, want 3", len(posts))
	}
	if posts[0].Id != "p2" || posts[1].Id != "p3" || posts[2].Id != "p1" {
		t.Errorf("wrong order: %s %s %s", posts[0].Id, posts[1].Id, posts[2].Id)
	}
}

func TestSortPosts_NilList(t *testing.T) {
	if got := SortPosts(nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestSortPosts_NilPosts(t *testing.T) {
	pl := &model.PostList{Posts: map[string]*model.Post{"p1": nil}}
	if posts := SortPosts(pl); len(posts) != 0 {
		t.Errorf("expected empty, got %d", len(posts))
	}
}
