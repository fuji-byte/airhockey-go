package dto

type MessageInput struct {
	// Typeは、クライアントからのメッセージの種類を識別するためのフィールド
	Type string `json:"type"`
	// ルームIDは、ユーザーがルームに参加する際に必要な情報
	RoomID string `json:"roomId"`
	// プレイヤーの位置情報は、ゲーム更新に必要な情報
	Position PositionInput `json:"position"`
}
