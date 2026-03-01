package telegram

type IncomingMessage struct {
	ID        int64
	From      string
	Text      string
	Timestamp int64
}
