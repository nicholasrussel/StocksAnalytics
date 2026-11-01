package notify

import "context"

type Message struct {
	Text string
}

type Notifier interface {
	Send(ctx context.Context, chatID int64, msg Message) error
}
