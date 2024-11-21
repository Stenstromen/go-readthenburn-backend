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
