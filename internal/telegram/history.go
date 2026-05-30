package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

// Message is a raw Telegram message returned from history or search calls.
type Message struct {
	ID        int
	Timestamp time.Time
	Text      string
}

// HistoryParams controls a messages.getHistory call.
type HistoryParams struct {
	Channel    *ResolvedChannel
	Limit      int
	OffsetID   int // 0 = start from newest
	OffsetDate int // Unix timestamp; return messages before this date (0 = no limit)
	MinDate    int // local filter lower bound
}

// SearchParams controls a messages.search call.
type SearchParams struct {
	Channel  *ResolvedChannel
	Query    string
	Limit    int
	OffsetID int
	MinDate  int
	MaxDate  int
}

// GetHistory fetches a page of channel history, newest first.
func GetHistory(ctx context.Context, api *tg.Client, p HistoryParams) ([]Message, error) {
	result, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:       p.Channel.InputPeer,
		OffsetID:   p.OffsetID,
		OffsetDate: p.OffsetDate,
		AddOffset:  0,
		Limit:      p.Limit,
		MaxID:      0,
		MinID:      0,
		Hash:       0,
	})
	if err != nil {
		return nil, fmt.Errorf("getHistory: %w", err)
	}
	msgs := extractMessages(result)
	if p.MinDate > 0 {
		filtered := msgs[:0]
		for _, m := range msgs {
			if int(m.Timestamp.Unix()) >= p.MinDate {
				filtered = append(filtered, m)
			}
		}
		msgs = filtered
	}
	return msgs, nil
}

// Search performs a text search within a channel.
func Search(ctx context.Context, api *tg.Client, p SearchParams) ([]Message, error) {
	result, err := api.MessagesSearch(ctx, &tg.MessagesSearchRequest{
		Peer:      p.Channel.InputPeer,
		Q:         p.Query,
		Filter:    &tg.InputMessagesFilterEmpty{},
		MinDate:   p.MinDate,
		MaxDate:   p.MaxDate,
		OffsetID:  p.OffsetID,
		AddOffset: 0,
		Limit:     p.Limit,
		MaxID:     0,
		MinID:     0,
		Hash:      0,
	})
	if err != nil {
		return nil, fmt.Errorf("messagesSearch: %w", err)
	}
	return extractMessages(result), nil
}

// GetMessage fetches a single message by ID from a channel or group.
func GetMessage(ctx context.Context, api *tg.Client, ch *ResolvedChannel, messageID int) (*Message, error) {
	ids := []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}}
	var result tg.MessagesMessagesClass
	var err error
	if ch.InputChannel != nil {
		result, err = api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: ch.InputChannel,
			ID:      ids,
		})
	} else {
		result, err = api.MessagesGetMessages(ctx, ids)
	}
	if err != nil {
		return nil, fmt.Errorf("getMessages id=%d: %w", messageID, err)
	}
	msgs := extractMessages(result)
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message %d not found", messageID)
	}
	return &msgs[0], nil
}

// extractMessages unwraps the union type returned by Telegram message API calls.
func extractMessages(v tg.MessagesMessagesClass) []Message {
	var raw []tg.MessageClass
	switch m := v.(type) {
	case *tg.MessagesMessages:
		raw = m.Messages
	case *tg.MessagesMessagesSlice:
		raw = m.Messages
	case *tg.MessagesChannelMessages:
		raw = m.Messages
	case *tg.MessagesMessagesNotModified:
		return nil
	}
	out := make([]Message, 0, len(raw))
	for _, item := range raw {
		if msg, ok := item.(*tg.Message); ok && msg.Message != "" {
			out = append(out, Message{
				ID:        msg.ID,
				Timestamp: time.Unix(int64(msg.Date), 0).UTC(),
				Text:      msg.Message,
			})
		}
	}
	return out
}
