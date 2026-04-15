package dto

type GameRoomOutput struct {
	// パックの座標と速度
	PuckX float64 `json:"PuckX"`
	PuckY float64 `json:"PuckY"`
	// それぞれのプレイヤーのスコア
	Score1 int `json:"Score1"`
	Score2 int `json:"Score2"`
	// ゲームの残り時間
	TimeLeftSec float32 `json:"TimeLeftSec"`
	Player1X    float64 `json:"Player1X"`
	Player1Y    float64 `json:"Player1Y"`
	Player2X    float64 `json:"Player2X"`
	Player2Y    float64 `json:"Player2Y"`
}
