package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// ResolveChannel resolves a reference to a channel. Supported forms, in order:
//   - "@username" (opens/returns the direct-message channel with that user)
//   - "email@host" (opens/returns the DM channel with that user)
//   - "~channel-name" (searches the user's teams for a matching channel)
//   - a 26-char ID (channel first, falling back to a user DM)
//   - "team/channel-name"
//   - "channel-name" (uses defaultTeam; falls back to a username DM)
//
// selfUserID is the ID of the authenticated user, required to open DM channels.
//
//nolint:gocognit,gocyclo,funlen // sequential six-form resolution chain
func (c *Client) ResolveChannel(ctx context.Context, ref, defaultTeam, selfUserID string) (*model.Channel, error) {
	// 1. @username
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

	// 2. Email (contains @ but doesn't start with @)
	if strings.Contains(ref, "@") {
		other, _, err := c.mm.GetUserByEmail(ctx, ref, "")
		if err != nil {
			return nil, fmt.Errorf("resolve email %q: %w", ref, err)
		}
		ch, _, err := c.mm.CreateDirectChannel(ctx, selfUserID, other.Id)
		if err != nil {
			return nil, fmt.Errorf("open DM with %q: %w", ref, err)
		}
		return ch, nil
	}

	// 3. ~channel (search across teams)
	if strings.HasPrefix(ref, "~") {
		name := strings.TrimPrefix(ref, "~")
		teams, _, err := c.mm.GetTeamsForUser(ctx, selfUserID, "")
		if err != nil {
			return nil, fmt.Errorf("list teams for ~%q: %w", name, err)
		}
		for _, t := range teams {
			ch, _, err := c.mm.GetChannelByName(ctx, name, t.Id, "")
			if err == nil && ch != nil {
				return ch, nil
			}
		}
		return nil, fmt.Errorf("channel ~%q not found in any team", name)
	}

	// 4. 26-char ID: channel first, user fallback
	if model.IsValidId(ref) {
		ch, _, err := c.mm.GetChannel(ctx, ref)
		if err == nil {
			return ch, nil
		}
		other, _, userErr := c.mm.GetUser(ctx, ref, "")
		if userErr != nil {
			return nil, fmt.Errorf("%q is not a channel or user", ref)
		}
		ch, _, dmErr := c.mm.CreateDirectChannel(ctx, selfUserID, other.Id)
		if dmErr != nil {
			return nil, fmt.Errorf("open DM: %w", dmErr)
		}
		return ch, nil
	}

	// 5. team/channel or bare word
	teamName, chanName := defaultTeam, ref
	if i := strings.IndexByte(ref, '/'); i >= 0 {
		teamName, chanName = ref[:i], ref[i+1:]
	}
	if teamName == "" {
		name, err := c.soleTeamName(ctx, selfUserID)
		if err != nil {
			return nil, err
		}
		teamName = name
	}

	team, _, err := c.mm.GetTeamByName(ctx, teamName, "")
	if err != nil {
		return nil, fmt.Errorf("resolve team %q: %w", teamName, err)
	}
	ch, _, err := c.mm.GetChannelByName(ctx, chanName, team.Id, "")
	if err == nil {
		return ch, nil
	}

	// 6. Bare word: channel not found → try user by username
	// Only meaningful when ref is a single word (no /).
	if !strings.Contains(ref, "/") {
		other, _, userErr := c.mm.GetUserByUsername(ctx, ref, "")
		if userErr != nil {
			return nil, fmt.Errorf("resolve channel %q: %w", ref, err)
		}
		dmCh, _, dmErr := c.mm.CreateDirectChannel(ctx, selfUserID, other.Id)
		if dmErr != nil {
			return nil, fmt.Errorf("open DM: %w", dmErr)
		}
		return dmCh, nil
	}
	return nil, fmt.Errorf("resolve channel %q: %w", ref, err)
}

// soleTeamName returns the name of the only team the user belongs to. It errors
// if the user is in zero or multiple teams (so the caller must qualify the
// channel as "team/channel").
func (c *Client) soleTeamName(ctx context.Context, selfUserID string) (string, error) {
	teams, _, err := c.mm.GetTeamsForUser(ctx, selfUserID, "")
	if err != nil {
		return "", fmt.Errorf("determine default team: %w", err)
	}
	switch len(teams) {
	case 0:
		return "", fmt.Errorf("you are not a member of any team")
	case 1:
		return teams[0].Name, nil
	default:
		names := make([]string, 0, len(teams))
		for _, t := range teams {
			names = append(names, t.Name)
		}
		return "", fmt.Errorf("multiple teams; qualify the channel as team/channel (teams: %s)", strings.Join(names, ", "))
	}
}
