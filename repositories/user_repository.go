package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"math/rand"
	"sync"
	"time"

	"main/dto"
	"main/models"

	"github.com/google/uuid"
)

type IMemoryRepository interface {
	CreateUser(user *models.User) (*models.User, error)
	DeleteUser(clientId string) error
	UserNum() int
	GetAllUser() (*map[string]*models.User, error)
	GetUserByClientId(Id string) (*models.User, error)
	GetUsersByRoomId(Id string) (*map[string]*models.User, error)
	MakeRoom(room *models.GameRoom, user *models.User) (*models.GameRoom, error)
	GetRoom(roomId string) (*models.GameRoom, error)
	JoinRoom(roomId string, user *models.User) (*map[string]*models.User, error)
	LeaveRoom(user *models.User, room *models.GameRoom) error
	SetGame(room *models.GameRoom) error
	RunGame(signal chan string, userCh chan *models.GameRoom, ch chan *models.GameRoom, room *models.GameRoom) error
	// TestRoom(ch chan *models.GameRoom) (*models.GameRoom, error)
	SetCh(room *models.GameRoom, ch chan *models.GameRoom, userch chan *models.GameRoom, signal chan string) error
	SaveLogRoom(room *models.GameRoom) error
	CheckCells(cellConnFrom, cellConnTo string, room *models.GameRoom) error
	CompareCellId(clientId, cellId string, room *models.GameRoom) error
	AddCellConn(cellConnFrom, cellConnTo string, room *models.GameRoom) error
	DelCellConn(cellConnFrom, cellConnTo string, room *models.GameRoom) error
}

// 現状、すべての変数にmuがつくため、効率が良くない
type MemoryRepository struct {
	memoryUser     map[string]*models.User     // ユーザーID → ユーザー
	memoryGameRoom map[string]*models.GameRoom // ルームID → ゲームルーム
	mu             sync.Mutex
}

func NewMemoryRepository(memoryUser map[string]*models.User, memoryGameRoom map[string]*models.GameRoom) IMemoryRepository {
	return &MemoryRepository{memoryUser: memoryUser, memoryGameRoom: memoryGameRoom}
}

func (s *MemoryRepository) CreateUser(user *models.User) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.memoryUser[(*user).ID] != nil {
		return nil, errors.New("user already exists")
	}
	s.memoryUser[(*user).ID] = user
	return user, nil
}

func (s *MemoryRepository) DeleteUser(clientId string) error {
	user, err := s.GetUserByClientId(clientId)
	if user == nil || err != nil {
		return err
	}
	// room, _ := s.GetRoom(user.RoomID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if room := s.memoryGameRoom[user.RoomID]; room != nil {
		s.DeleteUserInRoom(room, clientId)
	}

	if user := s.memoryUser[clientId]; user == nil {
		return errors.New("user already deleted")
	}
	delete(s.memoryUser, clientId)
	return nil
}

// 呼び出し元関数でmutexをしておくこと、roomがnilでないか確認しておくこと
func (s *MemoryRepository) DeleteUserInRoom(room *models.GameRoom, clientId string) error {
	delete(room.Players, clientId)
	// delete(room.Observers, clientId)
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
	if clientId == room.HostPlayer.ID {
		//host譲渡をここに書く また、hostが変更されたらhostになった人に通知
		var firstUser *models.User
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

func (s *MemoryRepository) GetAllUser() (*map[string]*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.memoryUser == nil {
		return nil, errors.New("users not found")
	}
	return &s.memoryUser, nil
}

func (s *MemoryRepository) GetUserByClientId(Id string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.memoryUser[Id]
	if !ok {
		return nil, errors.New("invalid userId")
	}
	return user, nil
}

func (s *MemoryRepository) GetUsersByRoomId(Id string) (*map[string]*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	users, ok := s.memoryGameRoom[Id]
	if !ok || users.Players == nil {
		return nil, errors.New("nil pointer room")
	}
	return &users.Players, nil
}

func (s *MemoryRepository) MakeRoom(room *models.GameRoom, user *models.User) (*models.GameRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryGameRoom[(*room).ID] = room
	(*user).RoomID = (*room).ID
	return room, nil
}

func (s *MemoryRepository) JoinRoom(roomId string, user *models.User) (*map[string]*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := s.memoryGameRoom[roomId]
	if room == nil {
		return nil, errors.New("empty room")
	}
	tempUsers := make(map[string]*models.User)

	for k, v := range room.Players {
		tempUsers[k] = v
	}

	for k, v := range room.Observers {
		tempUsers[k] = v
	}
	(*room).Players[user.ID] = user
	(*user).RoomID = roomId
	return &tempUsers, nil
}

func (s *MemoryRepository) LeaveRoom(user *models.User, room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	//room != nil確認済み
	user.RoomID = "-1"
	return s.DeleteUserInRoom(room, user.ID)
}

func (s *MemoryRepository) GetRoom(roomId string) (*models.GameRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := s.memoryGameRoom[roomId]
	if room == nil {
		return nil, errors.New("empty room")
	}
	return room, nil
}

// ※呼び出し元関数でロック必須
func (s *MemoryRepository) makeCell(clientId string, room *models.GameRoom) (*models.Cell, error) {

	//画面サイズ定義
	screenWidth := 360.0
	screenHeight := 640.0
	margin := 20.0

	active := false
	if clientId != "" {
		active = true
	}

	cell := models.Cell{
		ID:       uuid.New().String(),
		PlayerID: clientId,
		Progress: 0,
		X:        rand.Float64()*(screenWidth-2*margin) + margin,
		Y:        rand.Float64()*(screenHeight-2*margin) + margin,
		Rank:     1,
		// Power:    1,
		Active: active,
	}

	room.Cells[cell.ID] = &cell

	return &cell, nil
}

// ゲーム開始前の試合管理
func (s *MemoryRepository) SetGame(room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room == nil {
		return errors.New("empty room")
	}

	//各プレイヤーのcell生産
	for key := range room.Players {
		_, err := s.makeCell(key, room)
		if err != nil {
			return err
		}
	}
	i := 0
	for i < rand.Intn(3)+1 {
		s.makeCell("", room)
		i++
	}

	room.Started = true

	return nil
}

// chの使い道無い　現在
func (s *MemoryRepository) RunGame(signal chan string, userCh chan *models.GameRoom, ch chan *models.GameRoom, room *models.GameRoom) error {
	updateTicker := time.NewTicker(30 * time.Millisecond)
	endTimer := time.After(time.Duration(room.TimeLeftSec) * time.Second)
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
			s.Update(room)
			s.Broadcast(room)
			// s.SaveLogRoom(room)
			// room, err := s.TestRoom(ch)
			// if err != nil {
			// 	return err
			// }
		case <-endTimer:
			// 120秒経過でルームを終了
			fmt.Println("ルームのタイムアウトにより終了します:", room.ID)
			//またはチャネルによて終了シグナルが出たとき
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

// func (s *MemoryRepository) TestRoom(ch chan *models.GameRoom) (*models.GameRoom, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	newRoom := <-ch
// 	room := s.memoryGameRoom[newRoom.ID]
// 	//ここでnewRoomとroomを比較し、異常がないか検知する。現時点でのチート対策はない
// 	return room, nil
// }

func (s *MemoryRepository) Update(room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	room.TimeLeftSec -= 0.03

	//すべてのcell(active)の管理
	for _, inner := range room.Cells {
		if inner.Active {
			inner.Progress += 1
			if inner.Progress >= 100 {
				inner.Progress = 0
				inner.Rank += 1
			}
		}
	}

	//接続しているcellの管理 k = cellConnFrom ,innerK = cellConnTo
	for k, inner := range room.CellConn {
		for innerK := range inner {
			inner[innerK] += 1
			if inner[innerK] >= 50 {
				inner[innerK] = 0
				cell := room.Cells[innerK]
				if cell.PlayerID == room.Cells[k].PlayerID {
					cell.Rank += 1
				} else {
					cell.Rank -= 1
					//cell dead
					if cell.Rank <= 0 {
						cell.PlayerID = room.Cells[k].PlayerID
						room.CellConn[innerK] = make(map[string]int)
						cell.Active = true
						cell.Rank = 0
					}
				}
			}
		}
	}

	return nil
}

func (s *MemoryRepository) SaveLogRoom(room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	//プロトタイプ完成後に実装。
	//データベースなどにjsonで予定
	return nil
}

func (s *MemoryRepository) Broadcast(room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sendMessage := &dto.GameRoomOutput{
		Cells:       room.Cells,
		CellConn:    room.CellConn,
		TimeLeftSec: room.TimeLeftSec,
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

func (s *MemoryRepository) SetCh(room *models.GameRoom, ch chan *models.GameRoom, userch chan *models.GameRoom, signal chan string) error {
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

// roomにcellConnFrom, cellConnToが存在しているか調べる
func (s *MemoryRepository) CheckCells(cellConnFrom, cellConnTo string, room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if room == nil {
		return errors.New("empty room")
	}
	if room.Cells[cellConnFrom] == nil || room.Cells[cellConnTo] == nil {
		return errors.New("invalid cellId")
	}
	return nil
}

// cellのプレイヤーIDとクライアントIDを比較
func (s *MemoryRepository) CompareCellId(clientId, cellId string, room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// if room == nil {
	// 	return errors.New("empty room")
	// }
	cell := (*room).Cells[cellId]
	//stringより、nilチェックなし
	if (*cell).PlayerID != clientId {
		return errors.New("you can't operate this cell")
	}
	return nil
}

func (s *MemoryRepository) AddCellConn(cellConnFrom, cellConnTo string, room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// if room == nil {
	// 	return errors.New("empty room")
	// }

	//cellConnFrom検証している前提
	if _, exists := room.CellConn[cellConnFrom][cellConnTo]; exists {
		return errors.New("cell conn already exists")
	}

	if room.Cells[cellConnFrom].Rank/10+1 <= len(room.CellConn[cellConnFrom]) {
		return errors.New("cell conn already max")
	}

	if _, exists := room.CellConn[cellConnTo][cellConnFrom]; exists {
		if room.Cells[cellConnFrom].PlayerID == room.Cells[cellConnTo].PlayerID {
			return errors.New("cycle is not permitted")
		}
	}

	newRoom := *room
	// CellConn のディープコピー
	newCellConn := make(map[string]map[string]int, len(room.CellConn))
	for k, v := range room.CellConn {
		innerCopy := make(map[string]int, len(v))
		maps.Copy(innerCopy, v) // 内側のマップをコピー
		newCellConn[k] = innerCopy
	}

	// 新しい接続を追加
	if _, ok := newCellConn[cellConnFrom]; !ok {
		newCellConn[cellConnFrom] = make(map[string]int)
	}
	newCellConn[cellConnFrom][cellConnTo] = 0
	newRoom.CellConn = newCellConn

	//検証後にroomに保存する
	select {
	case room.UserCh <- &newRoom:
		return nil
	default:
		return errors.New("user channel is blocked")
	}
}

func (s *MemoryRepository) DelCellConn(cellConnFrom, cellConnTo string, room *models.GameRoom) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// if room == nil {
	// 	return errors.New("empty room")
	// }
	if _, exists := room.CellConn[cellConnFrom][cellConnTo]; !exists {
		return errors.New("no such key exists")
	}

	newRoom := *room
	// CellConn のディープコピー
	newCellConn := make(map[string]map[string]int, len(room.CellConn))
	for k, v := range room.CellConn {
		innerCopy := make(map[string]int, len(v))
		maps.Copy(innerCopy, v) // 内側のマップをコピー
		newCellConn[k] = innerCopy
	}

	delete(newCellConn[cellConnFrom], cellConnTo)
	newRoom.CellConn = newCellConn

	select {
	case room.UserCh <- &newRoom:
		return nil
	default:
		return errors.New("user channel is blocked")
	}
}
