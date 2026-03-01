package session

import "time"

type Message struct {
	ID        int64
	From      string
	Text      string
	Timestamp int64
}

func NewMessage(id int64, from, text string) *Message {
	return &Message{
		ID:        id,
		From:      from,
		Text:      text,
		Timestamp: time.Now().Unix(),
	}
}
