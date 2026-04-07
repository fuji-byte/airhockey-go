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
	GetAllUsers() (map[string]*dto.User, error)
	GetUserByClientID(clientID string) (*dto.User, error)
	MakeRoom(clientID string) (string, error)
	GetUsersByRoomId(Id string) (map[string]*dto.User, error)
	JoinRoom(roomId string, user *dto.User) (map[string]*dto.User, error)
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

func (s *MemoryService) GetAllUsers() (map[string]*dto.User, error) {
	return s.memoryRepository.GetAllUsers()
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

	newRoom := &dto.GameRoom{
		ID:         roomId,
		RoomName:   "",
		Players:    players,
		Observers:  observers,
		HostPlayer: user,
		Started:    false,
		Width:      640,
		Height:     320,
	}

	gameState := dto.GameState{
		PuckX:       newRoom.Width / 2,
		PuckY:       newRoom.Height / 2,
		PuckSpeedX:  2,
		PuckSpeedY:  2,
		Player1X:    newRoom.Width / 4,
		Player1Y:    newRoom.Height / 2,
		Player2X:    3 * newRoom.Width / 4,
		Player2Y:    newRoom.Height / 2,
		Score1:      0,
		Score2:      0,
		TimeLeftSec: 30,
	}

	newRoom.GameState = gameState

	// user, err := s.GetUserByClientID(clientID)
	_, err = s.memoryRepository.MakeRoom(newRoom, user)
	if err != nil {
		return "", err
	}
	return roomId, nil
}

func (s *MemoryService) GetUsersByRoomId(Id string) (map[string]*dto.User, error) {
	return s.memoryRepository.GetUsersByRoomId(Id)
}

func (s *MemoryService) JoinRoom(roomId string, user *dto.User) (map[string]*dto.User, error) {
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

// ゲーム開始の処理
func (s *MemoryService) StartGame(user *dto.User) error {
	// hostかどうか、ほかにプレイヤーが一人以上いるか
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
	// ルーム処理
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
	if room_member[clientID] == nil {
		return errors.New("user not found in the room")
	}
	return s.memoryRepository.UpdatePlayerPosition(roomId, clientID, playerX, playerY)
}
