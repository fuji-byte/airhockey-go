package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"main/dto"
)

type IMemoryRepository interface {
	CreateUser(user *dto.User) (*dto.User, error)
	DeleteUser(clientID string) error
	UserNum() int
	GetAllUsers() (map[string]*dto.User, error)
	GetUserByClientID(Id string) (*dto.User, error)
	GetUsersByRoomId(Id string) (map[string]*dto.User, error)
	MakeRoom(room *dto.GameRoom, user *dto.User) (*dto.GameRoom, error)
	GetRoom(roomId string) (*dto.GameRoom, error)
	JoinRoom(roomId string, user *dto.User) (map[string]*dto.User, error)
	LeaveRoom(user *dto.User, room *dto.GameRoom) error
	SetGame(room *dto.GameRoom) error
	RunGame(signal chan string, userCh chan *dto.GameRoom, ch chan *dto.GameRoom, room *dto.GameRoom) error
	// TestRoom(ch chan *dto.GameRoom) (*dto.GameRoom, error)
	SetCh(room *dto.GameRoom, ch chan *dto.GameRoom, userch chan *dto.GameRoom, signal chan string) error
	// SaveLogRoom(room *dto.GameRoom) error
	UpdatePlayerPosition(roomId string, clientID string, playerX, playerY float64) error
}

// 現状、すべての変数にmuがつくため、効率が良くない
type MemoryRepository struct {
	memoryUser     map[string]*dto.User     // ユーザーID → ユーザー
	memoryGameRoom map[string]*dto.GameRoom // ルームID → ゲームルーム
	mu             sync.Mutex
}

func NewMemoryRepository(memoryUser map[string]*dto.User, memoryGameRoom map[string]*dto.GameRoom) IMemoryRepository {
	return &MemoryRepository{memoryUser: memoryUser, memoryGameRoom: memoryGameRoom}
}

func (s *MemoryRepository) CreateUser(user *dto.User) (*dto.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.memoryUser[(*user).ID] != nil {
		return nil, errors.New("user already exists")
	}
	s.memoryUser[(*user).ID] = user
	return user, nil
}

func (s *MemoryRepository) DeleteUser(clientID string) error {
	user, err := s.GetUserByClientID(clientID)
	if user == nil || err != nil {
		return err
	}
	// room, _ := s.GetRoom(user.RoomID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if room := s.memoryGameRoom[user.RoomID]; room != nil {
		s.DeleteUserInRoom(room, clientID)
	}

	if user := s.memoryUser[clientID]; user == nil {
		return errors.New("user already deleted")
	}
	delete(s.memoryUser, clientID)
	return nil
}

// 呼び出し元関数でmutexをしておくこと、roomがnilでないか確認しておくこと
func (s *MemoryRepository) DeleteUserInRoom(room *dto.GameRoom, clientID string) error {
	delete(room.Players, clientID)
	// delete(room.Observers, clientID)
	//ホストの変更もしくはルームの削除を行う
	//観戦者を含めるか
	if len(room.Players) <= 0 {
		select {
		case room.Signal <- "end":
			log.Println("ルーム終了シグナル送信:", room.ID)
		default:
			// 既に closed か、受信バッファが満杯など
			log.Println("signal チャネルに送れませんでした（既に閉じられているかブロック）:", room.ID)
		}
		delete(s.memoryGameRoom, room.ID)
		return nil
	}
	if clientID == room.HostPlayer.ID {
		//host譲渡をここに書く また、hostが変更されたらhostになった人に通知
		var firstUser *dto.User
		for _, v := range room.Players {
			firstUser = v
			break // 最初の要素でループを抜ける
		}
		s.memoryGameRoom[room.ID].HostPlayer = firstUser
		for _, v := range room.Players {
			v.Send(`{"type":"message","message":"ルームから退出しました。"}`)
		}
		firstUser.Send(`{"type":"host"}`)
		firstUser.Send(`{"type":"message","message":"このルームのホストになりました。"}`)
	}
	for _, v := range room.Players {
		v.Send(fmt.Sprintf(`{"type":"roomNum","message":%d}`, len(room.Players))) //playerの数だけ
	}
	for _, v := range room.Observers {
		v.Send(fmt.Sprintf(`{"type":"roomNum","message":%d}`, len(room.Players))) //playerの数だけ
	}
	return nil
}

func (s *MemoryRepository) UserNum() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.memoryUser == nil {
		log.Fatal("userNum Error")
	}
	userNum := len(s.memoryUser)
	return userNum
}

func (s *MemoryRepository) GetAllUsers() (map[string]*dto.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.memoryUser == nil {
		return nil, errors.New("users not found")
	}
	return s.memoryUser, nil
}

func (s *MemoryRepository) GetUserByClientID(Id string) (*dto.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.memoryUser[Id]
	if !ok {
		return nil, errors.New("invalid userId")
	}
	return user, nil
}

func (s *MemoryRepository) GetUsersByRoomId(Id string) (map[string]*dto.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, ok := s.memoryGameRoom[Id]
	if !ok || users.Players == nil {
		return nil, errors.New("nil pointer room")
	}
	return users.Players, nil
}

func (s *MemoryRepository) MakeRoom(room *dto.GameRoom, user *dto.User) (*dto.GameRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryGameRoom[(*room).ID] = room
	(*user).RoomID = (*room).ID
	return room, nil
}

func (s *MemoryRepository) JoinRoom(roomId string, user *dto.User) (map[string]*dto.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := s.memoryGameRoom[roomId]
	if room == nil {
		return nil, errors.New("empty room")
	}
	tempUsers := make(map[string]*dto.User)

	for k, v := range room.Players {
		tempUsers[k] = v
	}

	for k, v := range room.Observers {
		tempUsers[k] = v
	}
	(*room).Players[user.ID] = user
	(*user).RoomID = roomId
	return tempUsers, nil
}

func (s *MemoryRepository) LeaveRoom(user *dto.User, room *dto.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	//room != nil確認済み
	user.RoomID = "-1"
	return s.DeleteUserInRoom(room, user.ID)
}

func (s *MemoryRepository) GetRoom(roomId string) (*dto.GameRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := s.memoryGameRoom[roomId]
	if room == nil {
		return nil, errors.New("empty room")
	}
	return room, nil
}

// ゲーム開始前の試合管理
func (s *MemoryRepository) SetGame(room *dto.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room == nil {
		return errors.New("empty room")
	}

	room.Started = true

	return nil
}

// chの使い道無い　現在
func (s *MemoryRepository) RunGame(signal chan string, userCh chan *dto.GameRoom, ch chan *dto.GameRoom, room *dto.GameRoom) error {
	updateTicker := time.NewTicker(30 * time.Millisecond)
	endTimer := time.After(time.Duration(room.GameState.TimeLeftSec) * time.Second)
	defer updateTicker.Stop()
	// チャネル閉鎖は once で一回だけ

	closeOnce := sync.Once{}
	closeRoomChannels := func() {
		closeOnce.Do(func() {
			if room.Signal != nil {
				close(room.Signal)
			}
			if room.UserCh != nil {
				close(room.UserCh)
			}
			if room.Ch != nil {
				close(room.Ch)
			}
			s.mu.Lock()
			delete(s.memoryGameRoom, room.ID)
			s.mu.Unlock()
			log.Println("ルームチャネルを安全に閉じました:", room.ID)
		})
	}

	for {
		select {
		case newRoom, ok := <-userCh:
			if !ok {
				log.Println("userCh が閉じられました:", room.ID)
				closeRoomChannels()
				return nil
			}
			// ユーザー入力を反映
			// newRoom の情報で room を更新する
			*room = *newRoom
		case <-updateTicker.C:
			// 30msごとの処理（ゲームロジックなど）
			//ユーザーから送信された情報を基に、updateに一時的に構造体を作成し、30msごとに更新する
			//異常、チートな移動、変更がないか また、ここで変更をlogとして保存しておく
			//test検証後にアップデートする
			//ルームのプレイヤーが０になったらsavelog以外消す？
			// 再接続可能にするか
			//更新した部分だけ、プレイヤーに送信する　例　時間だけ更新、cellConnだけ更新
			s.update(room)

			s.Broadcast(room)
			// s.SaveLogRoom(room)
			// room, err := s.TestRoom(ch)
			// if err != nil {
			// 	return err
			// }
		case <-endTimer:
			// 120秒経過でルームを終了
			fmt.Println("ルームのタイムアウトにより終了します:", room.ID)
			// またはチャネルによて終了シグナルが出たとき
			signal <- "end"
		case msg, ok := <-signal:
			if !ok {
				log.Println("signal チャネルが閉じられました:", room.ID)
				closeRoomChannels()
				return nil
			}
			if msg == "end" {
				room.Started = false
				for _, v := range room.Players {
					v.RoomID = "-1"
					v.Send(`{"type":"gameSet"}`)
				}
				fmt.Println("send game set")
				closeRoomChannels()
				return nil
			}
		}
	}
}

// func (s *MemoryRepository) TestRoom(ch chan *dto.GameRoom) (*dto.GameRoom, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	newRoom := <-ch
// 	room := s.memoryGameRoom[newRoom.ID]
// 	//ここでnewRoomとroomを比較し、異常がないか検知する。現時点でのチート対策はない
// 	return room, nil
// }

// func (s *MemoryRepository) SaveLogRoom(room *dto.GameRoom) error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	//プロトタイプ完成後に実装。
// 	//データベースなどにjsonで予定
// 	return nil
// }

func (s *MemoryRepository) update(room *dto.GameRoom) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ゲームの進行（時間経過、パックの移動）
	room.GameState.TimeLeftSec -= 0.03
	room.GameState.PuckX += room.GameState.PuckSpeedX
	room.GameState.PuckY += room.GameState.PuckSpeedY
	// パックの加速
	maxSpeed := 20.0
	if (room.GameState.PuckSpeedX > 0 && room.GameState.PuckSpeedX < maxSpeed) || (room.GameState.PuckSpeedX < 0 && room.GameState.PuckSpeedX > -maxSpeed) {
		room.GameState.PuckSpeedX *= 1.01
	}
	if (room.GameState.PuckSpeedY > 0 && room.GameState.PuckSpeedY < maxSpeed) || (room.GameState.PuckSpeedY < 0 && room.GameState.PuckSpeedY > -maxSpeed) {
		room.GameState.PuckSpeedY *= 1.01
	}

	// パックとパドルの反射処理
	d := math.Sqrt((room.GameState.Player1X-room.GameState.PuckX)*(room.GameState.Player1X-room.GameState.PuckX) + (room.GameState.Player1Y-room.GameState.PuckY)*(room.GameState.Player1Y-room.GameState.PuckY))
	if d < 45 { //パドルの半径とパックの半径の合計
		// 反射の計算（単純な反転）
		room.GameState.PuckSpeedX *= -1
		room.GameState.PuckSpeedY *= -1
	}
	d = math.Sqrt((room.GameState.Player2X-room.GameState.PuckX)*(room.GameState.Player2X-room.GameState.PuckX) + (room.GameState.Player2Y-room.GameState.PuckY)*(room.GameState.Player2Y-room.GameState.PuckY))
	if d < 45 {
		room.GameState.PuckSpeedX *= -1
		room.GameState.PuckSpeedY *= -1
	}

	// スコア処理
	if room.GameState.PuckX <= 0 {
		room.GameState.Score2++
		room.GameState.PuckX, room.GameState.PuckY = room.Width/2, room.Height/2
		room.GameState.PuckSpeedX = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
		room.GameState.PuckSpeedY = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
	}
	if room.GameState.PuckX >= room.Width {
		room.GameState.Score1++
		room.GameState.PuckX, room.GameState.PuckY = room.Width/2, room.Height/2
		room.GameState.PuckSpeedX = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
		room.GameState.PuckSpeedY = (rand.Float64()*3 + 2) * float64(rand.Intn(2)*2-1)
	}

	// 壁の反射処理
	if room.GameState.PuckY <= 0 || room.GameState.PuckY >= room.Height {
		room.GameState.PuckSpeedY *= -1
	}

	// プレイヤーが相手のコートに入らないようにする
	if room.GameState.Player1X < 0 {
		room.GameState.Player1X = 0
	}
	if room.GameState.Player2X > room.Width {
		room.GameState.Player2X = room.Width
	}
	if room.GameState.Player1X > room.Width/2 {
		room.GameState.Player1X = room.Width / 2
	}
	if room.GameState.Player2X < room.Width/2 {
		room.GameState.Player2X = room.Width / 2
	}
}
func (s *MemoryRepository) Broadcast(room *dto.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sendMessage := &dto.GameRoomOutput{
		PuckX:       room.GameState.PuckX,
		PuckY:       room.GameState.PuckY,
		Player1X:    room.GameState.Player1X,
		Player1Y:    room.GameState.Player1Y,
		Player2X:    room.GameState.Player2X,
		Player2Y:    room.GameState.Player2Y,
		Score1:      room.GameState.Score1,
		Score2:      room.GameState.Score2,
		TimeLeftSec: room.GameState.TimeLeftSec,
	}
	jsonData, err := json.Marshal(*sendMessage)
	if err != nil {
		fmt.Println(err)
	}
	msg := fmt.Sprintf(`{"type":"gameUpdate","message":%s}`, string(jsonData))
	for _, v := range room.Players {
		v.Send(msg)
	}
	for _, v := range room.Observers {
		v.Send(msg)
	}
	return nil
}

// SetChはroomのチャネルをセットする関数。
func (s *MemoryRepository) SetCh(room *dto.GameRoom, ch chan *dto.GameRoom, userch chan *dto.GameRoom, signal chan string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if room == nil {
		return errors.New("empty room")
	}
	(*room).Ch = ch
	(*room).UserCh = userch
	(*room).Signal = signal
	return nil
}

func (s *MemoryRepository) UpdatePlayerPosition(roomId string, clientID string, playerX, playerY float64) error {
	room, err := s.GetRoom(roomId)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// プレイヤーが存在している場合かつ、clientIDが一致する場合に位置を更新
	if clientID == room.HostPlayer.ID {
		room.GameState.Player1X = playerX
		room.GameState.Player1Y = playerY
	} else {
		room.GameState.Player2X = playerX
		room.GameState.Player2Y = playerY
	}
	return nil
}
