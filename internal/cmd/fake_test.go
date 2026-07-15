package cmd

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/client"
)

// fakeAPI implements client.API for command tests. Fields let each test set
// return values; unset methods return zero values.
type fakeAPI struct {
	me         *model.User
	status     *model.Status
	userByName *model.User
	teams      []*model.Team
	channels   []*model.Channel
	users      []*model.User
	posts      *model.PostList
	thread     *model.PostList
	threads    *model.Threads
	created    *model.Post
	lastPost   *model.Post
	resolved   *model.Channel
	dmChannel  *model.Channel
	fileData   []byte
	fileInfo   *model.FileInfo
	fileInfos  []*model.FileInfo
	uploadResp *model.FileUploadResponse
	loggedOut  bool
	err        error
	statusCode int //nolint:unused // reserved for tests asserting status code extraction

	viewedChannel string
	readThread    string
	reacted       string
	unreacted     string
	reactions     []*model.Reaction
	patched       *model.Post
	deleted       string

	streamEvents chan client.WSEvent
	streamErrs   chan error
	streamErr    error
}

var _ client.API = (*fakeAPI)(nil)

func (f *fakeAPI) Login(context.Context, string, string) (*model.User, error) { return f.me, f.err }
func (f *fakeAPI) LoginWithMFA(context.Context, string, string, string) (*model.User, error) {
	return f.me, f.err
}
func (f *fakeAPI) Logout(context.Context) error                          { f.loggedOut = true; return f.err }
func (f *fakeAPI) Token() string                                         { return "faketoken" }
func (f *fakeAPI) SetToken(string)                                       {}
func (f *fakeAPI) Me(context.Context) (*model.User, error)               { return f.me, f.err }
func (f *fakeAPI) Status(context.Context, string) (*model.Status, error) { return f.status, f.err }
func (f *fakeAPI) UserByUsername(context.Context, string) (*model.User, error) {
	return f.userByName, f.err
}

func (f *fakeAPI) UserByEmail(context.Context, string) (*model.User, error) {
	return f.userByName, f.err
}

func (f *fakeAPI) UsersByIDs(context.Context, []string) ([]*model.User, error) {
	return f.users, f.err
}

func (f *fakeAPI) SearchUsers(context.Context, string, string, int) ([]*model.User, error) {
	return f.users, f.err
}

func (f *fakeAPI) TeamsForUser(context.Context, string) ([]*model.Team, error) {
	return f.teams, f.err
}

func (f *fakeAPI) Team(context.Context, string) (*model.Team, error) {
	if len(f.teams) > 0 {
		return f.teams[0], f.err
	}
	return nil, f.err
}

func (f *fakeAPI) ChannelsForUser(context.Context, string, string) ([]*model.Channel, error) {
	return f.channels, f.err
}

func (f *fakeAPI) Channel(_ context.Context, id string) (*model.Channel, error) {
	if f.resolved != nil && f.resolved.Id == id {
		return f.resolved, f.err
	}
	return nil, f.err
}

func (f *fakeAPI) SearchChannels(context.Context, string, string) ([]*model.Channel, error) {
	return f.channels, f.err
}

func (f *fakeAPI) CreateDirectChannel(context.Context, string, string) (*model.Channel, error) {
	return f.dmChannel, f.err
}

func (f *fakeAPI) CreatePost(_ context.Context, p *model.Post) (*model.Post, error) {
	f.lastPost = p
	return f.created, f.err
}

func (f *fakeAPI) Search(context.Context, string, string, bool) (*model.PostList, error) {
	return f.posts, f.err
}

func (f *fakeAPI) PostsForChannel(context.Context, string, int) (*model.PostList, error) {
	return f.posts, f.err
}

func (f *fakeAPI) PostsSince(context.Context, string, int64) (*model.PostList, error) {
	return f.posts, f.err
}

func (f *fakeAPI) PostThread(context.Context, string) (*model.PostList, error) {
	return f.thread, f.err
}

func (f *fakeAPI) UserThreads(context.Context, string, string, bool, int) (*model.Threads, error) {
	return f.threads, f.err
}

func (f *fakeAPI) UploadFile(context.Context, []byte, string, string) (*model.FileUploadResponse, error) {
	return f.uploadResp, f.err
}
func (f *fakeAPI) GetFile(context.Context, string) ([]byte, error) { return f.fileData, f.err }
func (f *fakeAPI) FileInfo(context.Context, string) (*model.FileInfo, error) {
	return f.fileInfo, f.err
}

func (f *fakeAPI) FileInfosForPost(context.Context, string) ([]*model.FileInfo, error) {
	return f.fileInfos, f.err
}

func (f *fakeAPI) ViewChannel(_ context.Context, _, channelID string) error {
	f.viewedChannel = channelID
	return f.err
}

func (f *fakeAPI) UpdateThreadRead(_ context.Context, _, _, threadID string) error {
	f.readThread = threadID
	return f.err
}

func (f *fakeAPI) SaveReaction(_ context.Context, _, _, emoji string) error {
	f.reacted = emoji
	return f.err
}

func (f *fakeAPI) DeleteReaction(_ context.Context, _, _, emoji string) error {
	f.unreacted = emoji
	return f.err
}

func (f *fakeAPI) ReactionsForPost(context.Context, string) ([]*model.Reaction, error) {
	return f.reactions, f.err
}

func (f *fakeAPI) PatchPost(_ context.Context, _ string, msg string) (*model.Post, error) {
	f.patched = &model.Post{Message: msg}
	if f.err != nil {
		return nil, f.err
	}
	return f.patched, nil
}

func (f *fakeAPI) DeletePost(_ context.Context, postID string) error {
	f.deleted = postID
	return f.err
}
func (f *fakeAPI) ServerURL() string { return "https://mm.example.com" }
func (f *fakeAPI) ResolveChannel(context.Context, string, string, string) (*model.Channel, error) {
	return f.resolved, f.err
}

func (f *fakeAPI) StreamPosts(context.Context) (<-chan client.WSEvent, <-chan error, error) {
	if f.streamErr != nil {
		return nil, nil, f.streamErr
	}
	if f.streamEvents == nil {
		f.streamEvents = make(chan client.WSEvent)
	}
	if f.streamErrs == nil {
		f.streamErrs = make(chan error, 1)
	}
	return f.streamEvents, f.streamErrs, nil
}
