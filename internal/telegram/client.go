package telegram

import (
	"github.com/gotd/td/telegram"
)

// ClientOptions holds everything needed to build an MTProto client.
type ClientOptions struct {
	APIID          int
	APIHash        string
	SessionStorage telegram.SessionStorage
}

// NewClient returns a configured gotd/td Telegram client.
func NewClient(opts ClientOptions) *telegram.Client {
	return telegram.NewClient(opts.APIID, opts.APIHash, telegram.Options{
		SessionStorage: opts.SessionStorage,
	})
}
