package services

import (
	"errors"

	"main/models"
	"main/repositories"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type IMemoryService interface {
	CreateUser(clientId string, conn *websocket.Conn) (*models.User, error)
	DeleteUser(clientId string) error
	UserNum() int
	GetAllUser() (*map[string]*models.User, error)
	GetUserByClientId(clientId string) (*models.User, error)
	MakeRoom(clientId string) (string, error)
	GetUsersByRoomId(Id string) (*map[string]*models.User, error)
	JoinRoom(roomId string, user *models.User) (*map[string]*models.User, error)
	LeaveRoom(user *models.User) error
	StartGame(user *models.User) error
	GetRoomInfo(roomId string) (*models.GameRoom, error)
	CellConn(userId, roomId, cellConnFrom, cellConnTo string) error
	DelCellConn(userId, roomId, cellConnFrom, cellConnTo string) error
}

type MemoryService struct {
	memoryRepository repositories.IMemoryRepository
}

func NewMemoryService(memoryRepository repositories.IMemoryRepository) IMemoryService {
	return &MemoryService{memoryRepository: memoryRepository}
}

func (s *MemoryService) CreateUser(clientId string, conn *websocket.Conn) (*models.User, error) {
	newUser := &models.User{ID: clientId, Name: "guest", IsOnline: true, Conn: conn, SendCh: make(chan []byte, 256), RoomID: "-1"}
	return s.memoryRepository.CreateUser(newUser)
}

func (s *MemoryService) DeleteUser(clientId string) error {
	return s.memoryRepository.DeleteUser(clientId)
}

func (s *MemoryService) UserNum() int {
	return s.memoryRepository.UserNum()
}

func (s *MemoryService) GetAllUser() (*map[string]*models.User, error) {
	return s.memoryRepository.GetAllUser()
}

func (s *MemoryService) GetUserByClientId(clientId string) (*models.User, error) {
	user, err := s.memoryRepository.GetUserByClientId(clientId)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *MemoryService) MakeRoom(clientId string) (string, error) {
	//clietIdのroomが存在していないか
	user, err := s.memoryRepository.GetUserByClientId(clientId)
	if err != nil {
		return "", err
	}
	if (*user).RoomID != "-1" {
		return "", errors.New("user already in a Room")
	}
	roomId := uuid.New().String()

	players := map[string]*models.User{
		clientId: user,
	}
	var observers map[string]*models.User
	newRoom := &models.GameRoom{
		ID:          roomId,
		RoomName:    "",
		Players:     players,
		Observers:   observers,
		HostPlayer:  user,
		Started:     false,
		TimeLeftSec: 5,
	}
	// user, err := s.GetUserByClientId(clientId)
	_, err = s.memoryRepository.MakeRoom(newRoom, user)
	if err != nil {
		return "", err
	}
	return roomId, nil
}

func (s *MemoryService) GetUsersByRoomId(Id string) (*map[string]*models.User, error) {
	return s.memoryRepository.GetUsersByRoomId(Id)
}

func (s *MemoryService) JoinRoom(roomId string, user *models.User) (*map[string]*models.User, error) {
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

func (s *MemoryService) LeaveRoom(user *models.User) error {
	room, err := s.memoryRepository.GetRoom(user.RoomID)
	if err != nil {
		return err
	}
	return s.memoryRepository.LeaveRoom(user, room)
}

// 設計がよくないので作り直し（ルームを読み込み、ここからルームを送信している）
func (s *MemoryService) StartGame(user *models.User) error {
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
	ch := make(chan *models.GameRoom)
	signal := make(chan string, 5)
	userch := make(chan *models.GameRoom, 10)
	err = s.memoryRepository.SetCh(room, ch, userch, signal)
	if err != nil {
		return err
	}
	go s.memoryRepository.RunGame(signal, userch, ch, room)
	return nil
}

func (s *MemoryService) GetRoomInfo(roomId string) (*models.GameRoom, error) {
	roomInfo, err := s.memoryRepository.GetRoom(roomId)
	if err != nil {
		return nil, err
	}
	return roomInfo, err
}

func (s *MemoryService) CellConn(userId, roomId, cellConnFrom, cellConnTo string) error {
	room, err := s.memoryRepository.GetRoom(roomId)
	if err != nil {
		return err
	}
	err = s.memoryRepository.CheckCells(cellConnFrom, cellConnTo, room)
	if err != nil {
		return err
	}
	err = s.memoryRepository.CompareCellId(userId, cellConnFrom, room)
	if err != nil {
		return err
	}
	err = s.memoryRepository.AddCellConn(cellConnFrom, cellConnTo, room)
	if err != nil {
		return err
	}
	return nil
}

func (s *MemoryService) DelCellConn(userId, roomId, cellConnFrom, cellConnTo string) error {
	room, err := s.memoryRepository.GetRoom(roomId)
	if err != nil {
		return err
	}
	err = s.memoryRepository.CheckCells(cellConnFrom, cellConnTo, room)
	if err != nil {
		return err
	}
	err = s.memoryRepository.CompareCellId(userId, cellConnFrom, room)
	if err != nil {
		return err
	}
	err = s.memoryRepository.DelCellConn(cellConnFrom, cellConnTo, room)
	if err != nil {
		return err
	}
	return nil
}
