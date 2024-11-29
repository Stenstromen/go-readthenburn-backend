package models

type Message struct {
	ID        string
	Content   string
	IV        string
	Encrypted string
}

type MessageRequest struct {
	Message string `json:"message"`
}

type MessageResponse struct {
	MsgID   string `json:"msgId,omitempty"`
	BurnMsg string `json:"burnMsg,omitempty"`
	Error   string `json:"error,omitempty"`
}

type DailyStats struct {
	Date          string `json:"date"`
	TotalMessages int    `json:"totalMessages"`
}

type StatsResponse struct {
	TotalMessages int          `json:"totalMessages"`
	History       []DailyStats `json:"history"`
}
