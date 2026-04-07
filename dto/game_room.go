package dto

// redisでキャッシュで保管予定（db実装後）
// on memory

// ゲームルーム全体の構造体
type GameRoom struct {
	ID         string `gorm:"primaryKey"`
	RoomName   string //interfaceでもよい.プレイヤーが指定するルーム番号
	Players    map[string]*User
	Observers  map[string]*User
	HostPlayer *User
	Started    bool //default false
	GameState  GameState
	Signal     chan string
	Ch         chan *GameRoom
	UserCh     chan *GameRoom
}

// 変化の激しいゲームの状態を表す構造体
type GameState struct {
	// パックの座標と速度
	PuckX      float64 `json:"PuckX"`
	PuckY      float64 `json:"PuckY"`
	PuckSpeedX float64 `json:"PuckSpeedX"`
	PuckSpeedY float64 `json:"PuckSpeedY"`
	// プレイヤーの座標
	Player1X float64 `json:"Player1X"`
	Player1Y float64 `json:"Player1Y"`
	Player2X float64 `json:"Player2X"`
	Player2Y float64 `json:"Player2Y"`
	// それぞれのプレイヤーのスコア
	Score1 int `json:"Score1"`
	Score2 int `json:"Score2"`
	// ゲームの残り時間
	TimeLeftSec float32 `json:"TimeLeftSec"`
}
