package dto

type MessageInput struct {
	Type string `json:"type"`
	// Option   string   `json:"option"`
	RoomID string `json:"roomId"`
	// Message string `json:"message"`
	// Error    error    `json:"err"`
	Position PositionInput `json:"position"`
}
