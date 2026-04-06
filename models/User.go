package models

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// on memory
type User struct {
	ID       string //clientId　一応primaryKey
	Name     string
	IsOnline bool
	Conn     *websocket.Conn
	RoomID   string
	once     sync.Once
	SendCh   chan []byte
}

func (u *User) StartWriter() {
	for msg := range u.SendCh {
		if err := u.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			fmt.Println("Write error:", err)
			break // 書き込みエラーが出たらループ終了
		}
	}
}

// Send は SendCh にメッセージを送る
func (u *User) Send(msg string) {
	select {
	case u.SendCh <- []byte(msg):
	default:
		fmt.Println("SendCh is full, dropping message")
	}
}

func (u *User) Cleanup() {
	u.once.Do(func() {
		if u.Conn != nil {
			_ = u.Conn.Close()
		}
		// close は panic の元になるので、Send 側は送らない設計にするか、
		// どうしてもここで閉じるなら recover を使う
		close(u.SendCh)
	})
}
