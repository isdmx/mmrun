package client

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

// API is the Mattermost boundary used by commands. It is an interface so
// commands can be tested against a fake.
type API interface {
	Login(ctx context.Context, loginID, password string) (*model.User, error)
	LoginWithMFA(ctx context.Context, loginID, password, mfa string) (*model.User, error)
	Token() string
	SetToken(token string)
	Me(ctx context.Context) (*model.User, error)
	Status(ctx context.Context, userID string) (*model.Status, error)
	TeamsForUser(ctx context.Context, userID string) ([]*model.Team, error)
	ChannelsForUser(ctx context.Context, teamID, userID string) ([]*model.Channel, error)
	CreatePost(ctx context.Context, post *model.Post) (*model.Post, error)
	Search(ctx context.Context, teamID, terms string, orSearch bool) (*model.PostList, error)
	PostsForChannel(ctx context.Context, channelID string, perPage int) (*model.PostList, error)
	UploadFile(ctx context.Context, data []byte, channelID, filename string) (*model.FileUploadResponse, error)
	GetFile(ctx context.Context, fileID string) ([]byte, error)
	FileInfosForPost(ctx context.Context, postID string) ([]*model.FileInfo, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	ServerURL() string
	ResolveChannel(ctx context.Context, ref, defaultTeam string) (*model.Channel, error)
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

func (c *Client) Me(ctx context.Context) (*model.User, error) {
	u, _, err := c.mm.GetMe(ctx, "")
	return u, err
}

func (c *Client) Status(ctx context.Context, userID string) (*model.Status, error) {
	s, _, err := c.mm.GetUserStatus(ctx, userID, "")
	return s, err
}

func (c *Client) TeamsForUser(ctx context.Context, userID string) ([]*model.Team, error) {
	t, _, err := c.mm.GetTeamsForUser(ctx, userID, "")
	return t, err
}

func (c *Client) ChannelsForUser(ctx context.Context, teamID, userID string) ([]*model.Channel, error) {
	ch, _, err := c.mm.GetChannelsForTeamForUser(ctx, teamID, userID, false, "")
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

func (c *Client) UploadFile(ctx context.Context, data []byte, channelID, filename string) (*model.FileUploadResponse, error) {
	r, _, err := c.mm.UploadFile(ctx, data, channelID, filename)
	return r, err
}

func (c *Client) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	b, _, err := c.mm.GetFile(ctx, fileID)
	return b, err
}

func (c *Client) FileInfosForPost(ctx context.Context, postID string) ([]*model.FileInfo, error) {
	fi, _, err := c.mm.GetFileInfosForPost(ctx, postID, "")
	return fi, err
}

func (c *Client) RevokeSession(ctx context.Context, userID, sessionID string) error {
	_, err := c.mm.RevokeSession(ctx, userID, sessionID)
	return err
}
