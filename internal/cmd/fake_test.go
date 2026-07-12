package cmd

import (
	"context"

	"github.com/dmitriev/mmrun/internal/client"
	"github.com/mattermost/mattermost/server/public/model"
)

// fakeAPI implements client.API for command tests. Fields let each test set
// return values; unset methods return zero values.
type fakeAPI struct {
	me         *model.User
	status     *model.Status
	teams      []*model.Team
	channels   []*model.Channel
	posts      *model.PostList
	created    *model.Post
	resolved   *model.Channel
	fileData   []byte
	fileInfos  []*model.FileInfo
	uploadResp *model.FileUploadResponse
	err        error
}

var _ client.API = (*fakeAPI)(nil)

func (f *fakeAPI) Login(context.Context, string, string) (*model.User, error) { return f.me, f.err }
func (f *fakeAPI) LoginWithMFA(context.Context, string, string, string) (*model.User, error) {
	return f.me, f.err
}
func (f *fakeAPI) Token() string                                         { return "faketoken" }
func (f *fakeAPI) SetToken(string)                                       {}
func (f *fakeAPI) Me(context.Context) (*model.User, error)               { return f.me, f.err }
func (f *fakeAPI) Status(context.Context, string) (*model.Status, error) { return f.status, f.err }
func (f *fakeAPI) TeamsForUser(context.Context, string) ([]*model.Team, error) {
	return f.teams, f.err
}
func (f *fakeAPI) ChannelsForUser(context.Context, string, string) ([]*model.Channel, error) {
	return f.channels, f.err
}
func (f *fakeAPI) CreatePost(context.Context, *model.Post) (*model.Post, error) {
	return f.created, f.err
}
func (f *fakeAPI) Search(context.Context, string, string, bool) (*model.PostList, error) {
	return f.posts, f.err
}
func (f *fakeAPI) PostsForChannel(context.Context, string, int) (*model.PostList, error) {
	return f.posts, f.err
}
func (f *fakeAPI) UploadFile(context.Context, []byte, string, string) (*model.FileUploadResponse, error) {
	return f.uploadResp, f.err
}
func (f *fakeAPI) GetFile(context.Context, string) ([]byte, error) { return f.fileData, f.err }
func (f *fakeAPI) FileInfosForPost(context.Context, string) ([]*model.FileInfo, error) {
	return f.fileInfos, f.err
}
func (f *fakeAPI) RevokeSession(context.Context, string, string) error { return f.err }
func (f *fakeAPI) ServerURL() string                                   { return "https://mm.example.com" }
func (f *fakeAPI) ResolveChannel(context.Context, string, string) (*model.Channel, error) {
	return f.resolved, f.err
}
