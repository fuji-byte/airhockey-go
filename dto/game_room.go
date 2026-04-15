package dto

// ゲームルーム全体の構造体
type GameRoom struct {
	ID       string
	RoomName string // interfaceでもよい.プレイヤーが指定するルーム番号
	// ホストプレイヤー、参加プレイヤー、観戦者
	HostPlayer *User
	Players    map[string]*User
	Observers  map[string]*User
	// ゲームの開始状態
	Started bool // default false
	// ゲームの状態を表す構造体
	GameState GameState
	// クリーンアップ用のチャネル
	Signal chan string
	Ch     chan *GameRoom
	UserCh chan *GameRoom
	// ゲームのフィールドサイズ
	Width  float64
	Height float64
}
