package client

import (
	"context"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// WSEvent is a minimal, decoded websocket event.
type WSEvent struct {
	Event string
	Data  map[string]any
}

// StreamPosts connects the websocket and emits events until ctx is cancelled,
// reconnecting with backoff on failure.
func (c *Client) StreamPosts(ctx context.Context) (<-chan WSEvent, <-chan error, error) {
	out := make(chan WSEvent)
	errs := make(chan error, 1)

	wsURL := c.mm.URL
	go func() {
		backoff := time.Second
		for {
			ws, err := model.NewWebSocketClient4(toWS(wsURL), c.mm.AuthToken)
			if err != nil {
				select {
				case <-time.After(backoff):
					backoff = minDur(backoff*2, 30*time.Second)
					continue
				case <-ctx.Done():
					return
				}
			}
			ws.Listen()
			backoff = time.Second
			for {
				select {
				case ev, ok := <-ws.EventChannel:
					if !ok {
						ws.Close()
						goto reconnect
					}
					out <- WSEvent{Event: string(ev.EventType()), Data: ev.GetData()}
				case <-ctx.Done():
					ws.Close()
					return
				}
			}
		reconnect:
			select {
			case <-time.After(backoff):
				backoff = minDur(backoff*2, 30*time.Second)
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, errs, nil
}

func toWS(httpURL string) string {
	if len(httpURL) > 5 && httpURL[:5] == "https" {
		return "wss" + httpURL[5:]
	}
	if len(httpURL) > 4 && httpURL[:4] == "http" {
		return "ws" + httpURL[4:]
	}
	return httpURL
}

func minDur(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
