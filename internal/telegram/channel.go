package telegram

import (
	"context"
	"fmt"
	"strconv"
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
	InputChannel *tg.InputChannel // nil for regular groups
	DisplayName  string
}

// ResolveChannel resolves a channel identifier to its Telegram peer representation.
// Accepts:
//   - @username or username (public channel/supergroup)
//   - numeric ID like -2247666802 (group or supergroup from web URL)
func ResolveChannel(ctx context.Context, api *tg.Client, identifier string) (*ResolvedChannel, error) {
	clean := strings.TrimPrefix(identifier, "@")

	// Numeric ID path
	if id, err := strconv.ParseInt(clean, 10, 64); err == nil {
		return resolveByID(ctx, api, id)
	}

	// Username path
	return resolveByUsername(ctx, api, clean)
}

func resolveByUsername(ctx context.Context, api *tg.Client, username string) (*ResolvedChannel, error) {
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving channel @%s: %w", username, err)
	}
	for _, chat := range resolved.Chats {
		if c, ok := chat.(*tg.Channel); ok {
			return &ResolvedChannel{
				InputPeer: &tg.InputPeerChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				},
				InputChannel: &tg.InputChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				},
				DisplayName: "@" + username,
			}, nil
		}
	}
	return nil, fmt.Errorf("channel @%s not found in resolved result", username)
}

func resolveByID(ctx context.Context, api *tg.Client, id int64) (*ResolvedChannel, error) {
	// Telegram web URLs use negative IDs for groups; the absolute value is the chat/channel ID.
	absID := id
	if absID < 0 {
		absID = -absID
	}

	// Search dialogs to find the peer with this ID and get the access hash.
	result, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching dialogs to resolve ID %d: %w", id, err)
	}

	chats := extractDialogChats(result)
	for _, chat := range chats {
		switch c := chat.(type) {
		case *tg.Channel:
			if c.ID == absID {
				name := c.Title
				if c.Username != "" {
					name = "@" + c.Username
				}
				return &ResolvedChannel{
					InputPeer: &tg.InputPeerChannel{
						ChannelID:  c.ID,
						AccessHash: c.AccessHash,
					},
					InputChannel: &tg.InputChannel{
						ChannelID:  c.ID,
						AccessHash: c.AccessHash,
					},
					DisplayName: name,
				}, nil
			}
		case *tg.Chat:
			if c.ID == absID {
				return &ResolvedChannel{
					InputPeer:   &tg.InputPeerChat{ChatID: c.ID},
					InputChannel: nil,
					DisplayName: c.Title,
				}, nil
			}
		}
	}

	// If not found in first page of dialogs, try resolving as a supergroup directly.
	// This works if the user is a member but the group isn't in recent dialogs.
	return nil, fmt.Errorf("chat ID %d not found in dialogs — make sure the account is a member of this group", id)
}

func extractDialogChats(v tg.MessagesDialogsClass) []tg.ChatClass {
	switch d := v.(type) {
	case *tg.MessagesDialogs:
		return d.Chats
	case *tg.MessagesDialogsSlice:
		return d.Chats
	case *tg.MessagesDialogsNotModified:
		return nil
	}
	return nil
}
