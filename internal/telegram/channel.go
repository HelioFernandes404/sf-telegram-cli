package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"
)

// StripAt removes a leading "@" from a channel username.
func StripAt(s string) string {
	return strings.TrimPrefix(s, "@")
}

// ResolvedChannel holds the resolved peer representation needed for API calls.
type ResolvedChannel struct {
	InputPeer    tg.InputPeerClass
	InputChannel *tg.InputChannel
}

// ResolveChannel resolves a channel username to its Telegram peer representation.
// username may include or omit the leading "@".
func ResolveChannel(ctx context.Context, api *tg.Client, username string) (*ResolvedChannel, error) {
	username = StripAt(username)
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving channel @%s: %w", username, err)
	}

	for _, chat := range resolved.Chats {
		switch c := chat.(type) {
		case *tg.Channel:
			return &ResolvedChannel{
				InputPeer: &tg.InputPeerChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				},
				InputChannel: &tg.InputChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("channel @%s not found in resolved result", username)
}
