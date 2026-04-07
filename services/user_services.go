package services

import (
	"errors"

	"main/dto"
	"main/repositories"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type IMemoryService interface {
	CreateUser(clientID string, conn *websocket.Conn) (*dto.User, error)
	DeleteUser(clientID string) error
	UserNum() int
	GetAllUser() (map[string]*dto.User, error)
	GetUserByClientID(clientID string) (*dto.User, error)
	MakeRoom(clientID string) (string, error)
	GetUsersByRoomId(Id string) (*map[string]*dto.User, error)
	JoinRoom(roomId string, user *dto.User) (*map[string]*dto.User, error)
	LeaveRoom(user *dto.User) error
	StartGame(user *dto.User) error
	GetRoomInfo(roomId string) (*dto.GameRoom, error)
	UpdatePlayerPosition(roomId string, clientID string, playerX, playerY float64) error
}

type MemoryService struct {
	memoryRepository repositories.IMemoryRepository
}

func NewMemoryService(memoryRepository repositories.IMemoryRepository) IMemoryService {
	return &MemoryService{memoryRepository: memoryRepository}
}

func (s *MemoryService) CreateUser(clientID string, conn *websocket.Conn) (*dto.User, error) {
	newUser := &dto.User{ID: clientID, Name: "guest", IsOnline: true, Conn: conn, SendCh: make(chan []byte, 256), RoomID: "-1"}
	return s.memoryRepository.CreateUser(newUser)
}

func (s *MemoryService) DeleteUser(clientID string) error {
	return s.memoryRepository.DeleteUser(clientID)
}

func (s *MemoryService) UserNum() int {
	return s.memoryRepository.UserNum()
}

func (s *MemoryService) GetAllUser() (map[string]*dto.User, error) {
	return s.memoryRepository.GetAllUser()
}

func (s *MemoryService) GetUserByClientID(clientID string) (*dto.User, error) {
	user, err := s.memoryRepository.GetUserByClientID(clientID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *MemoryService) MakeRoom(clientID string) (string, error) {
	//clietIdのroomが存在していないか
	user, err := s.memoryRepository.GetUserByClientID(clientID)
	if err != nil {
		return "", err
	}
	if (*user).RoomID != "-1" {
		return "", errors.New("user already in a Room")
	}
	roomId := uuid.New().String()

	players := map[string]*dto.User{
		clientID: user,
	}

	var observers map[string]*dto.User

	gameState := dto.GameState{
		PuckX:      0,
		PuckY:      0,
		PuckSpeedX: 0,
		PuckSpeedY: 0,
	}

	newRoom := &dto.GameRoom{
		ID:         roomId,
		RoomName:   "",
		Players:    players,
		Observers:  observers,
		HostPlayer: user,
		Started:    false,
		GameState:  gameState,
	}

	// user, err := s.GetUserByClientID(clientID)
	_, err = s.memoryRepository.MakeRoom(newRoom, user)
	if err != nil {
		return "", err
	}
	return roomId, nil
}

func (s *MemoryService) GetUsersByRoomId(Id string) (*map[string]*dto.User, error) {
	return s.memoryRepository.GetUsersByRoomId(Id)
}

func (s *MemoryService) JoinRoom(roomId string, user *dto.User) (*map[string]*dto.User, error) {
	if (*user).RoomID != "-1" {
		return nil, errors.New("user already in a Room")
	}
	// room, err := s.memoryRepository.GetRoom(roomId)
	tempRoom, err := s.memoryRepository.JoinRoom(roomId, user)
	if err != nil {
		return nil, err
	}
	// room.Players[user.ID] = user
	// (*user).RoomID = roomId
	return tempRoom, nil
}

func (s *MemoryService) LeaveRoom(user *dto.User) error {
	room, err := s.memoryRepository.GetRoom(user.RoomID)
	if err != nil {
		return err
	}
	return s.memoryRepository.LeaveRoom(user, room)
}

// 設計がよくないので作り直し（ルームを読み込み、ここからルームを送信している）
func (s *MemoryService) StartGame(user *dto.User) error {
	//hostかどうか、ほかにプレイヤーが一人以上いるか
	roomId := user.RoomID
	room, err := s.memoryRepository.GetRoom(roomId)
	if err != nil {
		return err
	}
	if room.Started {
		return errors.New("the room is always started")
	}
	if user.ID != room.HostPlayer.ID {
		return errors.New("the user is not host")
	}
	if len(room.Players) <= 1 {
		return errors.New("the room doesn't exist member")
	}
	//ルーム処理
	err = s.memoryRepository.SetGame(room)
	if err != nil {
		return err
	}
	ch := make(chan *dto.GameRoom)
	signal := make(chan string, 5)
	userch := make(chan *dto.GameRoom, 10)
	err = s.memoryRepository.SetCh(room, ch, userch, signal)
	if err != nil {
		return err
	}
	go s.memoryRepository.RunGame(signal, userch, ch, room)
	return nil
}

func (s *MemoryService) GetRoomInfo(roomId string) (*dto.GameRoom, error) {
	roomInfo, err := s.memoryRepository.GetRoom(roomId)
	if err != nil {
		return nil, err
	}
	return roomInfo, err
}

func (s *MemoryService) UpdatePlayerPosition(roomId string, clientID string, playerX, playerY float64) error {
	// roomIdにclientIDのユーザーが存在しているか
	room_member, err := s.memoryRepository.GetUsersByRoomId(roomId)
	if err != nil {
		return err
	}
	if (*room_member)[clientID] == nil {
		return errors.New("user not found in the room")
	}
	return s.memoryRepository.UpdatePlayerPosition(roomId, clientID, playerX, playerY)
}
