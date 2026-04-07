package dto

import (
	"encoding/json"
)

type TypeInput struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type MessageInput struct {
	Type     string   `json:"type"`
	Option   string   `json:"option"`
	RoomID   string   `json:"roomId"`
	Message  string   `json:"message"`
	Error    error    `json:"err"`
	GameRoom GameRoom `json:"gameRoom"`
}

// type MessageOutput struct {
// 	Type    string `json:"type"`
// 	Option  string `json:"option"`
// 	RoomID  string `json:"roomId"`
// 	Message string `json:"message"`
// 	Error   error  `json:"err"`
// }

// type GameRoomInput struct {
// 	Type         string `json:"type"`
// 	CellConnFrom string `json:"cFrom"`
// 	CellConnTo   string `json:"cTo"`
// }

// type GameRoomOutput struct {
// 	Cells       map[string]*models.Cell
// 	CellConn    map[string]map[string]int
// 	TimeLeftSec float32
// }
