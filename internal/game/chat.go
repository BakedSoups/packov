package game

import "time"

type ChatMessage struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	SenderID  PlayerID  `json:"sender_id"`
	Sender    string    `json:"sender"`
	Body      string    `json:"body"`
	SentAtUTC time.Time `json:"sent_at_utc"`
}
