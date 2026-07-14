// Package client is the sole boundary to the Mattermost server API. It wraps
// model.Client4 behind the API interface so commands can be tested against a
// fake.
package client

import (
	"context"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// API is the Mattermost boundary used by commands. It is an interface so
// commands can be tested against a fake.
type API interface {
	Login(ctx context.Context, loginID, password string) (*model.User, error)
	LoginWithMFA(ctx context.Context, loginID, password, mfa string) (*model.User, error)
	Logout(ctx context.Context) error
	Token() string
	SetToken(token string)
	Me(ctx context.Context) (*model.User, error)
	Status(ctx context.Context, userID string) (*model.Status, error)
	UserByUsername(ctx context.Context, username string) (*model.User, error)
	UserByEmail(ctx context.Context, email string) (*model.User, error)
	UsersByIDs(ctx context.Context, ids []string) ([]*model.User, error)
	SearchUsers(ctx context.Context, term, teamID string, limit int) ([]*model.User, error)
	TeamsForUser(ctx context.Context, userID string) ([]*model.Team, error)
	Team(ctx context.Context, teamID string) (*model.Team, error)
	ChannelsForUser(ctx context.Context, teamID, userID string) ([]*model.Channel, error)
	Channel(ctx context.Context, channelID string) (*model.Channel, error)
	SearchChannels(ctx context.Context, teamID, term string) ([]*model.Channel, error)
	CreateDirectChannel(ctx context.Context, userID1, userID2 string) (*model.Channel, error)
	CreatePost(ctx context.Context, post *model.Post) (*model.Post, error)
	Search(ctx context.Context, teamID, terms string, orSearch bool) (*model.PostList, error)
	PostsForChannel(ctx context.Context, channelID string, perPage int) (*model.PostList, error)
	PostsSince(ctx context.Context, channelID string, since int64) (*model.PostList, error)
	PostThread(ctx context.Context, postID string) (*model.PostList, error)
	UserThreads(ctx context.Context, userID, teamID string, unread bool, pageSize int) (*model.Threads, error)
	UploadFile(ctx context.Context, data []byte, channelID, filename string) (*model.FileUploadResponse, error)
	GetFile(ctx context.Context, fileID string) ([]byte, error)
	FileInfo(ctx context.Context, fileID string) (*model.FileInfo, error)
	FileInfosForPost(ctx context.Context, postID string) ([]*model.FileInfo, error)
	ViewChannel(ctx context.Context, userID, channelID string) error
	UpdateThreadRead(ctx context.Context, userID, teamID, threadID string) error
	SaveReaction(ctx context.Context, postID, userID, emojiName string) error
	DeleteReaction(ctx context.Context, postID, userID, emojiName string) error
	ReactionsForPost(ctx context.Context, postID string) ([]*model.Reaction, error)
	PatchPost(ctx context.Context, postID, msg string) (*model.Post, error)
	DeletePost(ctx context.Context, postID string) error
	ServerURL() string
	ResolveChannel(ctx context.Context, ref, defaultTeam, selfUserID string) (*model.Channel, error)
	StreamPosts(ctx context.Context) (<-chan WSEvent, <-chan error, error)
}

// Client wraps model.Client4 and satisfies API.
type Client struct {
	mm *model.Client4
}

// NewWithToken builds a Client authenticated by an existing token.
func NewWithToken(serverURL, token string) *Client {
	mm := model.NewAPIv4Client(serverURL)
	mm.SetToken(token)
	return &Client{mm: mm}
}

// New builds an unauthenticated Client (for the login flow).
func New(serverURL string) *Client {
	return &Client{mm: model.NewAPIv4Client(serverURL)}
}

func (c *Client) ServerURL() string { return c.mm.URL }
func (c *Client) Token() string     { return c.mm.AuthToken }
func (c *Client) SetToken(t string) { c.mm.SetToken(t) }

func (c *Client) Login(ctx context.Context, loginID, password string) (*model.User, error) {
	u, _, err := c.mm.Login(ctx, loginID, password)
	return u, err
}

func (c *Client) LoginWithMFA(ctx context.Context, loginID, password, mfa string) (*model.User, error) {
	u, _, err := c.mm.LoginWithMFA(ctx, loginID, password, mfa)
	return u, err
}

// Logout terminates (revokes) the current session token server-side.
func (c *Client) Logout(ctx context.Context) error {
	_, err := c.mm.Logout(ctx)
	return err
}

func (c *Client) Me(ctx context.Context) (*model.User, error) {
	u, _, err := c.mm.GetMe(ctx, "")
	return u, err
}

func (c *Client) Status(ctx context.Context, userID string) (*model.Status, error) {
	s, _, err := c.mm.GetUserStatus(ctx, userID, "")
	return s, err
}

func (c *Client) UserByUsername(ctx context.Context, username string) (*model.User, error) {
	u, _, err := c.mm.GetUserByUsername(ctx, username, "")
	return u, err
}

func (c *Client) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	u, _, err := c.mm.GetUserByEmail(ctx, email, "")
	return u, err
}

func (c *Client) UsersByIDs(ctx context.Context, ids []string) ([]*model.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	u, _, err := c.mm.GetUsersByIds(ctx, ids)
	return u, err
}

func (c *Client) SearchUsers(ctx context.Context, term, teamID string, limit int) ([]*model.User, error) {
	u, _, err := c.mm.SearchUsers(ctx, &model.UserSearch{Term: term, TeamId: teamID, Limit: limit})
	return u, err
}

func (c *Client) TeamsForUser(ctx context.Context, userID string) ([]*model.Team, error) {
	t, _, err := c.mm.GetTeamsForUser(ctx, userID, "")
	return t, err
}

func (c *Client) Team(ctx context.Context, teamID string) (*model.Team, error) {
	t, _, err := c.mm.GetTeam(ctx, teamID, "")
	return t, err
}

func (c *Client) ChannelsForUser(ctx context.Context, teamID, userID string) ([]*model.Channel, error) {
	ch, _, err := c.mm.GetChannelsForTeamForUser(ctx, teamID, userID, false, "")
	return ch, err
}

func (c *Client) Channel(ctx context.Context, channelID string) (*model.Channel, error) {
	ch, _, err := c.mm.GetChannel(ctx, channelID)
	return ch, err
}

func (c *Client) SearchChannels(ctx context.Context, teamID, term string) ([]*model.Channel, error) {
	ch, _, err := c.mm.SearchChannels(ctx, teamID, &model.ChannelSearch{Term: term})
	return ch, err
}

func (c *Client) CreateDirectChannel(ctx context.Context, userID1, userID2 string) (*model.Channel, error) {
	ch, _, err := c.mm.CreateDirectChannel(ctx, userID1, userID2)
	return ch, err
}

func (c *Client) CreatePost(ctx context.Context, post *model.Post) (*model.Post, error) {
	p, _, err := c.mm.CreatePost(ctx, post)
	return p, err
}

func (c *Client) Search(ctx context.Context, teamID, terms string, orSearch bool) (*model.PostList, error) {
	pl, _, err := c.mm.SearchPosts(ctx, teamID, terms, orSearch)
	return pl, err
}

func (c *Client) PostsForChannel(ctx context.Context, channelID string, perPage int) (*model.PostList, error) {
	pl, _, err := c.mm.GetPostsForChannel(ctx, channelID, 0, perPage, "", false, false)
	return pl, err
}

func (c *Client) PostsSince(ctx context.Context, channelID string, since int64) (*model.PostList, error) {
	pl, _, err := c.mm.GetPostsSince(ctx, channelID, since, false)
	return pl, err
}

func (c *Client) PostThread(ctx context.Context, postID string) (*model.PostList, error) {
	pl, _, err := c.mm.GetPostThread(ctx, postID, "", false)
	return pl, err
}

// UserThreads returns the threads the user follows in a team, most recently
// updated first.
func (c *Client) UserThreads(ctx context.Context, userID, teamID string, unread bool, pageSize int) (*model.Threads, error) {
	opts := model.GetUserThreadsOpts{Unread: unread}
	if pageSize > 0 {
		opts.PageSize = uint64(pageSize)
	}
	th, _, err := c.mm.GetUserThreads(ctx, userID, teamID, opts)
	return th, err
}

func (c *Client) UploadFile(ctx context.Context, data []byte, channelID, filename string) (*model.FileUploadResponse, error) {
	r, _, err := c.mm.UploadFile(ctx, data, channelID, filename)
	return r, err
}

func (c *Client) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	b, _, err := c.mm.GetFile(ctx, fileID)
	return b, err
}

func (c *Client) FileInfo(ctx context.Context, fileID string) (*model.FileInfo, error) {
	fi, _, err := c.mm.GetFileInfo(ctx, fileID)
	return fi, err
}

func (c *Client) FileInfosForPost(ctx context.Context, postID string) ([]*model.FileInfo, error) {
	fi, _, err := c.mm.GetFileInfosForPost(ctx, postID, "")
	return fi, err
}

func (c *Client) ViewChannel(ctx context.Context, userID, channelID string) error {
	_, _, err := c.mm.ViewChannel(ctx, userID, &model.ChannelView{
		ChannelId:     channelID,
		PrevChannelId: channelID,
	})
	return err
}

func (c *Client) UpdateThreadRead(ctx context.Context, userID, teamID, threadID string) error {
	_, _, err := c.mm.UpdateThreadReadForUser(ctx, userID, teamID, threadID, time.Now().UnixMilli())
	return err
}

func (c *Client) SaveReaction(ctx context.Context, postID, userID, emojiName string) error {
	r := &model.Reaction{UserId: userID, PostId: postID, EmojiName: emojiName}
	_, _, err := c.mm.SaveReaction(ctx, r)
	return err
}

func (c *Client) DeleteReaction(ctx context.Context, postID, userID, emojiName string) error {
	r := &model.Reaction{UserId: userID, PostId: postID, EmojiName: emojiName}
	_, err := c.mm.DeleteReaction(ctx, r)
	return err
}

func (c *Client) ReactionsForPost(ctx context.Context, postID string) ([]*model.Reaction, error) {
	rr, _, err := c.mm.GetReactions(ctx, postID)
	return rr, err
}

func (c *Client) PatchPost(ctx context.Context, postID, msg string) (*model.Post, error) {
	p, _, err := c.mm.PatchPost(ctx, postID, &model.PostPatch{Message: &msg})
	return p, err
}

func (c *Client) DeletePost(ctx context.Context, postID string) error {
	_, err := c.mm.DeletePost(ctx, postID)
	return err
}
