package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// ResolveChannel resolves a reference to a channel. Supported forms:
//   - "team/channel-name"
//   - "channel-name" (uses defaultTeam)
//   - a 26-char channel ID
//
// DM support (@username) is added in the post/read tasks.
func (c *Client) ResolveChannel(ctx context.Context, ref, defaultTeam string) (*model.Channel, error) {
	if model.IsValidId(ref) {
		ch, _, err := c.mm.GetChannel(ctx, ref)
		return ch, err
	}

	teamName, chanName := defaultTeam, ref
	if i := strings.IndexByte(ref, '/'); i >= 0 {
		teamName, chanName = ref[:i], ref[i+1:]
	}
	if teamName == "" {
		return nil, fmt.Errorf("no team specified for channel %q and no default team set", ref)
	}

	team, _, err := c.mm.GetTeamByName(ctx, teamName, "")
	if err != nil {
		return nil, fmt.Errorf("resolve team %q: %w", teamName, err)
	}
	ch, _, err := c.mm.GetChannelByName(ctx, chanName, team.Id, "")
	if err != nil {
		return nil, fmt.Errorf("resolve channel %q: %w", chanName, err)
	}
	return ch, nil
}
