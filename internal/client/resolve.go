package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// ResolveChannel resolves a reference to a channel. Supported forms:
//   - "@username" (opens/returns the direct-message channel with that user)
//   - "team/channel-name"
//   - "channel-name" (uses defaultTeam)
//   - a 26-char channel ID
//
// selfUserID is the ID of the authenticated user, required to open DM channels.
func (c *Client) ResolveChannel(ctx context.Context, ref, defaultTeam, selfUserID string) (*model.Channel, error) {
	if strings.HasPrefix(ref, "@") {
		username := strings.TrimPrefix(ref, "@")
		other, _, err := c.mm.GetUserByUsername(ctx, username, "")
		if err != nil {
			return nil, fmt.Errorf("resolve user %q: %w", username, err)
		}
		ch, _, err := c.mm.CreateDirectChannel(ctx, selfUserID, other.Id)
		if err != nil {
			return nil, fmt.Errorf("open DM with %q: %w", username, err)
		}
		return ch, nil
	}

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
