package server

import (
	"bufio"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"
)

const (
	defaultRoomCode    = ""
	onlineCodeLength   = 5
	onlineCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type Server struct {
	listener     net.Listener
	mu           sync.Mutex
	rooms        map[string]*matchRoom
	clients      map[string]*serverClient
	closed       chan struct{}
	closeOnce    sync.Once
	nextClientID uint64
}

type matchRoom struct {
	code         string
	name         string
	state        sim.GameState
	hostClientID string
	teamOwners   map[sim.Team]string
	currentInput map[sim.Team]sim.InputFrame
}

type serverClient struct {
	id       string
	team     sim.Team
	roomCode string
	conn     net.Conn
	encoder  *json.Encoder
	mu       sync.Mutex
}

type roomSnapshot struct {
	roomCode string
	snapshot sim.GameState
	clients  []*serverClient
}

func Listen(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		listener: listener,
		rooms:    make(map[string]*matchRoom),
		clients:  make(map[string]*serverClient),
		closed:   make(chan struct{}),
	}, nil
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) PlayerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.ensureRoomLocked(defaultRoomCode, "").teamOwners)
}

func (s *Server) SetLobbyColors(homeColor, awayColor sim.TeamColor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := s.ensureRoomLocked(defaultRoomCode, "")
	room.state.HomeColor = homeColor
	room.state.AwayColor = awayColor
}

func (s *Server) Serve() error {
	go s.tickLoop()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return nil
			default:
			}
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		err = s.listener.Close()
		s.mu.Lock()
		clients := make([]*serverClient, 0, len(s.clients))
		for _, client := range s.clients {
			clients = append(clients, client)
		}
		s.mu.Unlock()
		for _, client := range clients {
			_ = client.conn.Close()
		}
	})
	return err
}

func (s *Server) handleConn(conn net.Conn) {
	decoder := json.NewDecoder(bufio.NewReader(conn))
	encoder := json.NewEncoder(conn)

	var join netcode.Message
	if err := decoder.Decode(&join); err != nil {
		_ = conn.Close()
		return
	}
	switch join.Kind {
	case netcode.MessageRoomListRequest:
		_ = encoder.Encode(netcode.Message{Kind: netcode.MessageRoomList, Rooms: s.listRooms()})
		_ = conn.Close()
		return
	case netcode.MessageJoinRequest:
		// Continue with the multiplayer handshake.
	default:
		_ = encoder.Encode(netcode.Message{Kind: netcode.MessageError, Error: "expected join_request or room_list_request"})
		_ = conn.Close()
		return
	}

	client, accepted, errMessage := s.registerClient(conn, join)
	if errMessage != "" {
		_ = encoder.Encode(netcode.Message{Kind: netcode.MessageError, Error: errMessage})
		_ = conn.Close()
		return
	}
	client.encoder = encoder
	if err := client.send(accepted); err != nil {
		s.unregisterClient(client.id)
		return
	}
	go s.readClientLoop(client, decoder)
}

func (s *Server) registerClient(conn net.Conn, join netcode.Message) (*serverClient, netcode.Message, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, errMessage := s.roomForJoinLocked(join)
	if errMessage != "" {
		return nil, netcode.Message{}, errMessage
	}
	team, ok := s.nextOpenTeamLocked(room)
	if !ok {
		return nil, netcode.Message{}, "room already has two players"
	}
	clientID := fmt.Sprintf("client-%d", atomic.AddUint64(&s.nextClientID, 1))
	client := &serverClient{id: clientID, team: team, roomCode: room.code, conn: conn}
	if room.hostClientID == "" {
		room.hostClientID = clientID
	}
	isHost := room.hostClientID == clientID

	s.clients[clientID] = client
	room.teamOwners[team] = clientID
	room.currentInput[team] = sim.InputFrame{ClientID: clientID, Team: team}
	snapshot := cloneGameState(room.state)
	accepted := netcode.Message{
		Kind:     netcode.MessageJoinAccepted,
		MatchID:  matchIDForRoom(room.code),
		RoomCode: room.code,
		RoomName: room.name,
		ClientID: clientID,
		Team:     team,
		TickRate: sim.TickRate,
		State:    &snapshot,
		Host:     isHost,
	}
	return client, accepted, ""
}

func (s *Server) roomForJoinLocked(join netcode.Message) (*matchRoom, string) {
	roomCode := normalizeRoomCode(join.RoomCode)
	if join.CreateRoom {
		if roomCode == defaultRoomCode {
			generated, err := s.uniqueRoomCodeLocked()
			if err != nil {
				return nil, "unable to create room"
			}
			roomCode = generated
		}
		if _, exists := s.rooms[roomCode]; exists {
			return nil, "room already exists"
		}
		return s.createRoomLocked(roomCode, join.RoomName), ""
	}
	if roomCode == defaultRoomCode {
		return s.ensureRoomLocked(defaultRoomCode, ""), ""
	}
	room, ok := s.rooms[roomCode]
	if !ok {
		return nil, "room not found"
	}
	return room, ""
}

func (s *Server) ensureRoomLocked(roomCode, roomName string) *matchRoom {
	if room, ok := s.rooms[roomCode]; ok {
		if roomName != "" && room.name == "" {
			room.name = strings.TrimSpace(roomName)
		}
		return room
	}
	return s.createRoomLocked(roomCode, roomName)
}

func (s *Server) createRoomLocked(roomCode, roomName string) *matchRoom {
	room := &matchRoom{
		code:         roomCode,
		name:         strings.TrimSpace(roomName),
		state:        sim.NewMultiplayerGameState(),
		hostClientID: "",
		teamOwners:   make(map[sim.Team]string),
		currentInput: make(map[sim.Team]sim.InputFrame),
	}
	s.rooms[roomCode] = room
	return room
}

func (s *Server) uniqueRoomCodeLocked() (string, error) {
	for attempts := 0; attempts < 32; attempts++ {
		code, err := randomRoomCode()
		if err != nil {
			return "", err
		}
		if _, exists := s.rooms[code]; !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("unable to allocate unique room code")
}

func (s *Server) nextOpenTeamLocked(room *matchRoom) (sim.Team, bool) {
	if room == nil {
		return sim.TeamNone, false
	}
	if _, ok := room.teamOwners[sim.TeamHome]; !ok {
		return sim.TeamHome, true
	}
	if _, ok := room.teamOwners[sim.TeamAway]; !ok {
		return sim.TeamAway, true
	}
	return sim.TeamNone, false
}

func (s *Server) unregisterClient(clientID string) {
	s.mu.Lock()
	client, ok := s.clients[clientID]
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(s.clients, clientID)
	if room, ok := s.rooms[client.roomCode]; ok {
		if owner, ok := room.teamOwners[client.team]; ok && owner == clientID {
			delete(room.teamOwners, client.team)
			delete(room.currentInput, client.team)
		}
		if len(room.teamOwners) == 0 && client.roomCode != defaultRoomCode {
			delete(s.rooms, client.roomCode)
		}
	}
	s.mu.Unlock()
	_ = client.conn.Close()
}

func (s *Server) readClientLoop(client *serverClient, decoder *json.Decoder) {
	defer s.unregisterClient(client.id)
	for {
		var message netcode.Message
		if err := decoder.Decode(&message); err != nil {
			return
		}
		if message.Kind != netcode.MessageInputFrame {
			continue
		}
		input := message.ToInputFrame()
		input.ClientID = client.id
		input.Team = client.team
		s.mu.Lock()
		if room, ok := s.rooms[client.roomCode]; ok {
			room.currentInput[client.team] = input
		}
		s.mu.Unlock()
	}
}

func (s *Server) tickLoop() {
	ticker := time.NewTicker(time.Second / sim.TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.closed:
			return
		case <-ticker.C:
			fanouts := s.stepAndSnapshot()
			for _, fanout := range fanouts {
				message := netcode.Message{
					Kind:     netcode.MessageSnapshot,
					MatchID:  matchIDForRoom(fanout.roomCode),
					RoomCode: fanout.roomCode,
					Tick:     fanout.snapshot.Tick,
					State:    &fanout.snapshot,
				}
				for _, client := range fanout.clients {
					if err := client.send(message); err != nil {
						s.unregisterClient(client.id)
					}
				}
			}
		}
	}
}

func (s *Server) stepAndSnapshot() []roomSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	fanouts := make([]roomSnapshot, 0, len(s.rooms))
	for roomCode, room := range s.rooms {
		inputs := make([]sim.InputFrame, 0, 2)
		for _, team := range []sim.Team{sim.TeamHome, sim.TeamAway} {
			clientID, ok := room.teamOwners[team]
			if !ok {
				continue
			}
			input := room.currentInput[team]
			input.ClientID = clientID
			input.Team = team
			input.Tick = room.state.Tick + 1
			inputs = append(inputs, input)
		}

		sim.Step(&room.state, inputs)
		snapshot := cloneGameState(room.state)
		clients := make([]*serverClient, 0, len(room.teamOwners))
		for _, team := range []sim.Team{sim.TeamHome, sim.TeamAway} {
			clientID, ok := room.teamOwners[team]
			if !ok {
				continue
			}
			if client, ok := s.clients[clientID]; ok {
				clients = append(clients, client)
			}
		}
		fanouts = append(fanouts, roomSnapshot{roomCode: roomCode, snapshot: snapshot, clients: clients})
	}
	return fanouts
}

func (c *serverClient) send(message netcode.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.encoder.Encode(message)
}

func cloneGameState(state sim.GameState) sim.GameState {
	copyState := state
	copyState.HomeSkaters = append([]sim.SkaterState(nil), state.HomeSkaters...)
	copyState.AwaySkaters = append([]sim.SkaterState(nil), state.AwaySkaters...)
	return copyState
}

func (s *Server) listRooms() []netcode.RoomSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms := make([]netcode.RoomSummary, 0, len(s.rooms))
	for roomCode, room := range s.rooms {
		if roomCode == defaultRoomCode {
			continue
		}
		name := strings.TrimSpace(room.name)
		if name == "" {
			name = "Room " + room.code
		}
		rooms = append(rooms, netcode.RoomSummary{Code: room.code, Name: name, Players: len(room.teamOwners), Capacity: 2})
	}

	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].Joinable() != rooms[j].Joinable() {
			return rooms[i].Joinable()
		}
		if rooms[i].Name != rooms[j].Name {
			return rooms[i].Name < rooms[j].Name
		}
		return rooms[i].Code < rooms[j].Code
	})
	return rooms
}

func normalizeRoomCode(code string) string {
	trimmed := strings.TrimSpace(strings.ToUpper(code))
	if trimmed == "" {
		return defaultRoomCode
	}
	buffer := make([]byte, 0, onlineCodeLength)
	for _, r := range trimmed {
		if len(buffer) >= onlineCodeLength {
			break
		}
		if strings.ContainsRune(onlineCodeAlphabet, r) {
			buffer = append(buffer, byte(r))
		}
	}
	return string(buffer)
}

func randomRoomCode() (string, error) {
	buffer := make([]byte, onlineCodeLength)
	if _, err := crand.Read(buffer); err != nil {
		return "", err
	}
	code := make([]byte, onlineCodeLength)
	for index, value := range buffer {
		code[index] = onlineCodeAlphabet[int(value)%len(onlineCodeAlphabet)]
	}
	return string(code), nil
}

func matchIDForRoom(roomCode string) string {
	if roomCode == defaultRoomCode {
		return "match-1"
	}
	return "room-" + roomCode
}
