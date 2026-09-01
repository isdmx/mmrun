package client

import (
	"context"
	"sync/atomic"

	"github.com/mattermost/mattermost/server/public/model"
)

// FakeAPI implements API for tests. Fields let each test set return values;
// unset methods return zero values.
type FakeAPI struct {
	Me_         *model.User
	Status_     *model.Status
	UserByName_ *model.User
	Teams_      []*model.Team
	Channels_   []*model.Channel
	Users_      []*model.User
	Posts_      *model.PostList
	Post_       *model.Post
	Thread_     *model.PostList
	Threads_    *model.Threads
	Members_    model.ChannelMembers
	Created_    *model.Post
	LastPost_   *model.Post
	Resolved_   *model.Channel
	DMChannel_  *model.Channel
	FileData_   []byte
	FileInfo_   *model.FileInfo
	FileInfos_  []*model.FileInfo
	UploadResp_ *model.FileUploadResponse
	LoggedOut_  bool
	Err         error

	ViewedChannel_ string
	ReadThread_    string
	Followed_      string
	Unfollowed_    string
	Reacted_       string
	Unreacted_     string
	Reactions_     []*model.Reaction
	Patched_       *model.Post
	Deleted_       string
	Pinned_        string
	Unpinned_      string
	Statuses_      []*model.Status

	StreamEvents_ chan WSEvent
	StreamErrs_   chan error
	StreamErr_    error

	PinnedPosts_   *model.PostList
	ChannelStats_  *model.ChannelStats
	ChannelUnread_ *model.ChannelUnread
	FlaggedPosts_  *model.PostList
	Bots_          []*model.Bot
	PostFlagged_   string
	SearchTerms_   string
	SearchCalls_   int

	TeamsForUserCalls_      int
	ChannelsForUserCalls_   int
	UserThreadsCalls_       int
	PostThreadCalls_        int32
	PostThreadPerPage_      int
	PostsForChannelPerPage_ int
	UsersByIDsArgs_         []string
	UserThreadsSince        int64
	ChannelMembersChannel   string
}

var _ API = (*FakeAPI)(nil)

func (f *FakeAPI) Login(context.Context, string, string) (*model.User, error) {
	return f.Me_, f.Err
}

func (f *FakeAPI) LoginWithMFA(context.Context, string, string, string) (*model.User, error) {
	return f.Me_, f.Err
}
func (f *FakeAPI) Logout(context.Context) error { f.LoggedOut_ = true; return f.Err }
func (f *FakeAPI) Token() string                { return "faketoken" }
func (f *FakeAPI) SetToken(string)              {}
func (f *FakeAPI) Me(context.Context) (*model.User, error) {
	return f.Me_, f.Err
}

func (f *FakeAPI) Status(context.Context, string) (*model.Status, error) {
	return f.Status_, f.Err
}

func (f *FakeAPI) UserByUsername(context.Context, string) (*model.User, error) {
	return f.UserByName_, f.Err
}

func (f *FakeAPI) UserByEmail(context.Context, string) (*model.User, error) {
	return f.UserByName_, f.Err
}

func (f *FakeAPI) UsersByIDs(_ context.Context, ids []string) ([]*model.User, error) {
	f.UsersByIDsArgs_ = ids
	return f.Users_, f.Err
}

func (f *FakeAPI) SearchUsers(context.Context, string, string, int) ([]*model.User, error) {
	return f.Users_, f.Err
}

func (f *FakeAPI) TeamsForUser(context.Context, string) ([]*model.Team, error) {
	f.TeamsForUserCalls_++
	return f.Teams_, f.Err
}

func (f *FakeAPI) Team(context.Context, string) (*model.Team, error) {
	if len(f.Teams_) > 0 {
		return f.Teams_[0], f.Err
	}
	return nil, f.Err
}

func (f *FakeAPI) ChannelsForUser(context.Context, string, string) ([]*model.Channel, error) {
	f.ChannelsForUserCalls_++
	return f.Channels_, f.Err
}

func (f *FakeAPI) Channel(_ context.Context, id string) (*model.Channel, error) {
	if f.Resolved_ != nil && f.Resolved_.Id == id {
		return f.Resolved_, f.Err
	}
	return nil, f.Err
}

func (f *FakeAPI) ChannelMembers(_ context.Context, channelID string, _, _ int) (model.ChannelMembers, error) {
	f.ChannelMembersChannel = channelID
	return f.Members_, f.Err
}

func (f *FakeAPI) SearchChannels(context.Context, string, string) ([]*model.Channel, error) {
	return f.Channels_, f.Err
}

func (f *FakeAPI) CreateDirectChannel(context.Context, string, string) (*model.Channel, error) {
	return f.DMChannel_, f.Err
}

func (f *FakeAPI) CreatePost(_ context.Context, p *model.Post) (*model.Post, error) {
	f.LastPost_ = p
	if f.Created_ != nil {
		return f.Created_, f.Err
	}
	return p, f.Err
}

func (f *FakeAPI) Search(_ context.Context, _ string, terms string, _ bool, _ int, _ int) (*model.PostList, error) {
	f.SearchTerms_ = terms
	f.SearchCalls_++
	return f.Posts_, f.Err
}

func (f *FakeAPI) PinnedPosts(context.Context, string) (*model.PostList, error) {
	return f.PinnedPosts_, f.Err
}

func (f *FakeAPI) ChannelStats(context.Context, string) (*model.ChannelStats, error) {
	return f.ChannelStats_, f.Err
}

func (f *FakeAPI) ChannelUnread(context.Context, string, string) (*model.ChannelUnread, error) {
	return f.ChannelUnread_, f.Err
}

func (f *FakeAPI) FlaggedPosts(context.Context, string, string, int, int) (*model.PostList, error) {
	return f.FlaggedPosts_, f.Err
}

func (f *FakeAPI) FlagPost(_ context.Context, id string) error { f.PostFlagged_ = id; return f.Err }

func (f *FakeAPI) UnflagPost(_ context.Context, _ string) error { f.PostFlagged_ = ""; return f.Err }
func (f *FakeAPI) Bots(context.Context) ([]*model.Bot, error)   { return f.Bots_, f.Err }
func (f *FakeAPI) PostsForChannel(_ context.Context, _ string, perPage int) (*model.PostList, error) {
	f.PostsForChannelPerPage_ = perPage
	return f.Posts_, f.Err
}

func (f *FakeAPI) PostsSince(context.Context, string, int64) (*model.PostList, error) {
	return f.Posts_, f.Err
}

func (f *FakeAPI) GetPost(context.Context, string) (*model.Post, error) {
	return f.Post_, f.Err
}

func (f *FakeAPI) PostThread(_ context.Context, _ string) (*model.PostList, error) {
	atomic.AddInt32(&f.PostThreadCalls_, 1)
	return f.Thread_, f.Err
}

func (f *FakeAPI) PostThreadPaged(_ context.Context, _ string, perPage int) (*model.PostList, error) {
	f.PostThreadPerPage_ = perPage
	return f.Thread_, f.Err
}

func (f *FakeAPI) UserThreads(_ context.Context, _, _ string, _ bool, _ int, since int64) (*model.Threads, error) {
	f.UserThreadsCalls_++
	f.UserThreadsSince = since
	return f.Threads_, f.Err
}

func (f *FakeAPI) UploadFile(context.Context, []byte, string, string) (*model.FileUploadResponse, error) {
	return f.UploadResp_, f.Err
}
func (f *FakeAPI) GetFile(context.Context, string) ([]byte, error) { return f.FileData_, f.Err }
func (f *FakeAPI) FileInfo(context.Context, string) (*model.FileInfo, error) {
	return f.FileInfo_, f.Err
}

func (f *FakeAPI) FileInfosForPost(context.Context, string) ([]*model.FileInfo, error) {
	return f.FileInfos_, f.Err
}

func (f *FakeAPI) ViewChannel(_ context.Context, _, channelID string) error {
	f.ViewedChannel_ = channelID
	return f.Err
}

func (f *FakeAPI) UpdateThreadRead(_ context.Context, _, _, threadID string) error {
	f.ReadThread_ = threadID
	return f.Err
}

func (f *FakeAPI) FollowThread(_ context.Context, _, _, threadID string) error {
	f.Followed_ = threadID
	return f.Err
}

func (f *FakeAPI) UnfollowThread(_ context.Context, _, _, threadID string) error {
	f.Unfollowed_ = threadID
	return f.Err
}

func (f *FakeAPI) SaveReaction(_ context.Context, _, _, emoji string) error {
	f.Reacted_ = emoji
	return f.Err
}

func (f *FakeAPI) DeleteReaction(_ context.Context, _, _, emoji string) error {
	f.Unreacted_ = emoji
	return f.Err
}

func (f *FakeAPI) ReactionsForPost(context.Context, string) ([]*model.Reaction, error) {
	return f.Reactions_, f.Err
}

func (f *FakeAPI) PatchPost(_ context.Context, _ string, msg string) (*model.Post, error) {
	f.Patched_ = &model.Post{Message: msg}
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Patched_, f.Err
}

func (f *FakeAPI) DeletePost(_ context.Context, postID string) error {
	f.Deleted_ = postID
	return f.Err
}
func (f *FakeAPI) ServerURL() string { return "https://mm.example.com" }
func (f *FakeAPI) ResolveChannel(context.Context, string, string, string) (*model.Channel, error) {
	return f.Resolved_, f.Err
}

func (f *FakeAPI) StreamPosts(context.Context) (<-chan WSEvent, <-chan error, error) {
	if f.StreamErr_ != nil {
		return nil, nil, f.StreamErr_
	}
	if f.StreamEvents_ == nil {
		f.StreamEvents_ = make(chan WSEvent)
	}
	if f.StreamErrs_ == nil {
		f.StreamErrs_ = make(chan error, 1)
	}
	return f.StreamEvents_, f.StreamErrs_, nil
}
func (f *FakeAPI) PinPost(_ context.Context, id string) error   { f.Pinned_ = id; return f.Err }
func (f *FakeAPI) UnpinPost(_ context.Context, id string) error { f.Unpinned_ = id; return f.Err }
func (f *FakeAPI) UsersStatuses(context.Context, []string) ([]*model.Status, error) {
	return f.Statuses_, f.Err
}
func (f *FakeAPI) UpdateStatus(_ context.Context, _, _ string) error          { return f.Err }
func (f *FakeAPI) UpdateCustomStatus(_ context.Context, _, _, _ string) error { return f.Err }
