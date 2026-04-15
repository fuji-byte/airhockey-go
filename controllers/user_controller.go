package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"main/dto"
	"main/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type IMemoryController interface {
	HandleWebSocket(ctx *gin.Context)
}

type MemoryController struct {
	service services.IMemoryService
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	// HandshakeTimeout	ハンドシェイクのタイムアウト時間	通常 5秒〜10秒程度が一般的
	// ReadBufferSize / WriteBufferSize	I/Oバッファのサイズ（バイト単位）	多数接続時や大規模アプリで最適化可能
	// WriteBufferPool	書き込みバッファのプール	接続数が多く、GC削減したいときに有効
	// Subprotocols	WebSocketのサブプロトコル（例: chat, json, etc.）	クライアントとプロトコルネゴシエート可
	// CheckOrigin	オリジンチェック（CORS対応）	本番ではここでOriginを検証すべき
	// EnableCompression	per-message圧縮を有効化（RFC 7692）	クライアントがサポートしていれば圧縮される
}

func NewMemoryController(service services.IMemoryService) IMemoryController {
	return &MemoryController{service: service}
}

func (c *MemoryController) HandleWebSocket(ctx *gin.Context) {

	// HTTP 接続を WebSocket にアップグレード
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		fmt.Println("WebSocket 接続エラー:", err)
		return
	}
	defer conn.Close()

	// 新しいクライアントを登録
	clientID := uuid.New().String()
	user, err := c.service.CreateUser(clientID, conn)
	go user.StartWriter()
	if err != nil {
		log.Fatal("CreateUser Error")
	}
	userNum := c.userNumControll()
	fmt.Println("クライアントが接続しました.現在", userNum, "人")

	//websocket接続切断時の処理
	defer func() {
		//user情報削除、または一定時間保持
		//room情報 dto.GameRoomの情報変更,dto.User[clientID]の削除
		//データベース使うならuser.IsOnline falseにする

		// user, err := c.service.GetUserByClientID(clientID)
		// if err != nil {
		// 	fmt.Println("user取得エラー")
		// }
		//errは使わない　ユーザーがroomが存在していない場合があるから
		users, _ := c.service.GetUsersByRoomId(user.RoomID)
		err := c.service.DeleteUser(clientID)
		if err != nil {
			log.Fatal("DeleteUser Error")
		}
		c.userNumControll()
		if users != nil {
			broadcast(users, "roomNum", "message", len(users))
			fmt.Println(users)
		}
		fmt.Println("切断後処理完了:", clientID)
	}()

	//接続状態時の処理
	for {
		_, msg, err := user.Conn.ReadMessage()
		if err != nil {
			fmt.Println("接続が切断されました:", err)
			user.Cleanup()
			// pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			break
		}
		fmt.Printf("受信メッセージ: %s\n", msg)
		fmt.Printf("メッセージを受信しました\n")
		var receivedMsg dto.MessageInput
		if err := json.Unmarshal(msg, &receivedMsg); err != nil {
			fmt.Println("MessageInput の解析に失敗:", err)
			continue
		}
		switch receivedMsg.Type {
		case "makeRoom":
			roomId, err := c.service.MakeRoom(clientID)
			if err != nil {
				fmt.Println("MakeRoom Error:", err)
				msg := `{"type":"errorMessage","message":"ルームを作成できませんでした。","error": "make a room Error"}`
				user.Send(msg)
				continue
			}
			user.Send(fmt.Sprintf(`{"type":"roomId","message":"%v"}`, roomId))
			user.Send(`{"type":"roomNum", "message":"1"}`)
			user.Send(`{"type":"host"}`)
			continue
		case "joinRoom":
			user, err := c.service.GetUserByClientID(clientID)
			if err != nil {
				fmt.Println("no users:", err)
				user.Send(`{"type":"errorMessage","message":"ユーザーを取得できませんでした。","error": "couldn't get the users Error"}`)
				continue
			}
			tempUsers, err := c.service.JoinRoom(receivedMsg.RoomID, user)
			if err != nil {
				fmt.Println("join room Error:", err)
				user.Send(`{"type":"errorMessage","message":"ルームに参加できませんでした。","error": "join in the room Error"}`)
				continue
			}
			room, err := c.service.GetUsersByRoomId(receivedMsg.RoomID)
			if err != nil {
				fmt.Println("no users:", err)
				user.Send(`{"type":"errorMessage","message":"ルームメンバーを取得できませんでした。","error": "couldn't get the room member(s) Error"}`)
				continue
			}
			msg := fmt.Sprintf(`%vが参加しました`, user.Name)
			broadcast(room, "roomNum", "message", len(room))
			broadcast(tempUsers, "message", "message", msg)
			user.Send(fmt.Sprintf(`{"type":"roomId","message":"%v"}`, user.RoomID))
			continue
		case "leaveRoom":
			user, err := c.service.GetUserByClientID(clientID)
			if err != nil {
				fmt.Println("no users:", err)
				user.Send(`{"type":"errorMessage","message":"ユーザーを取得できませんでした。","error": "couldn't get the users Error"}`)
				continue
			}
			err = c.service.LeaveRoom(user)
			if err != nil {
				user.Send(`{"type":"errorMessage","message":"ルームの退出に失敗しました","error": "failed to leave the room Error"}`)
				continue
			}
			user.Send(`{"type":"leaveRoom", "message":"ルームを退出しました"}`)
			continue
		case "match":
			//host playerがmatchを送信したら
			user, err := c.service.GetUserByClientID(clientID)
			if err != nil {
				fmt.Println("no users:", err)
				user.Send(`{"type":"errorMessage","message":"ユーザーを取得できませんでした。","error": "couldn't get the users Error"}`)
				continue
			}
			err = c.service.StartGame(user)
			if err != nil {
				user.Send(`{"type":"errorMessage","message":"ゲームのスタートに失敗しました","error": "failed to start the game or invalid host Error"}`)
				continue
			}
			room, err := c.service.GetUsersByRoomId(user.RoomID)
			if err != nil {
				fmt.Println("no users:", err)
				user.Send(`{"type":"errorMessage","message":"ルームメンバーを取得できませんでした。","error": "couldn't get the room member(s) Error"}`)
				continue
			}

			for _, user := range room {
				if user == nil {
					fmt.Printf("send Message Error to user\n")
					continue
				}
				msg := fmt.Sprintf(`{"type":"%v", "%v": "%v"}`, "gameStart", "message", user.ID)
				user.Send(msg)
			}
			continue
		case "game":
			user, err := c.service.GetUserByClientID(clientID)
			if err != nil {
				fmt.Println("no users:", err)
				user.Send(`{"type":"errorMessage","message":"ユーザーを取得できませんでした。","error": "couldn't get the users Error"}`)
				continue
			}
			err = c.service.UpdatePlayerPosition(user.RoomID, clientID, receivedMsg.Position.PlayerX, receivedMsg.Position.PlayerY)
			if err != nil {
				fmt.Println("UpdatePlayerPosition Error:", err)
				user.Send(`{"type":"errorMessage","message":"プレイヤーの位置を更新できませんでした。","error": "failed to update player position Error"}`)
				continue
			}
			continue
		case "observe":
			//roomのobserverに追加
			//終わるまでか、観戦キャンセルされるまでずっとブロードキャスト
			continue
		case "reset":
			users, err := c.service.GetAllUsers()
			if err != nil {
				fmt.Println("no users", err)
				continue
			}
			user.Send(fmt.Sprintf(`{"type":"userNum","message":"%d"}`, len(users)))
			continue
		default:
			user.Send(`{"type":"errorMessage","message":"タイプが適切ではありません","error": "type Error"}`)
			continue
		}
	}
}

func (c *MemoryController) userNumControll() int {
	//s.memoryService.userNum()を取得して、User全員にブロードキャスト
	users, err := c.service.GetAllUsers()
	if err != nil {
		fmt.Println("no users", err)
		return -1
	}
	broadcast(users, "userNum", "message", len(users))
	return len(users)
}

// broadcast wants users map[string]*dto.User, messageType string, content(int float32 string)
func broadcast[T int | float32 | string](
	users map[string]*dto.User,
	messageType1 string,
	messageType2 string,
	content T,
) {
	message := fmt.Sprintf(`{"type":"%v", "%v": "%v"}`, messageType1, messageType2, content)
	// jsonBytes, err := json.Marshal(message)
	// if err != nil {
	// 	fmt.Println("Marshalエラー:", err)
	// 	return
	// }
	for _, user := range users {
		if user == nil {
			fmt.Printf("send Message Error to user\n")
		}
		user.Send(message)
	}
}
