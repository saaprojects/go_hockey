package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"
)

type Server struct {
	listener     net.Listener
	matchID      string
	state        sim.GameState
	mu           sync.Mutex
	clients      map[string]*serverClient
	teamOwners   map[sim.Team]string
	currentInput map[sim.Team]sim.InputFrame
	closed       chan struct{}
	closeOnce    sync.Once
	nextClientID uint64
}

type serverClient struct {
	id      string
	team    sim.Team
	conn    net.Conn
	encoder *json.Encoder
	mu      sync.Mutex
}

func Listen(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		listener:     listener,
		matchID:      "match-1",
		state:        sim.NewMultiplayerGameState(),
		clients:      make(map[string]*serverClient),
		teamOwners:   make(map[sim.Team]string),
		currentInput: make(map[sim.Team]sim.InputFrame),
		closed:       make(chan struct{}),
	}, nil
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) PlayerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.teamOwners)
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
	if join.Kind != netcode.MessageJoinRequest {
		_ = encoder.Encode(netcode.Message{Kind: netcode.MessageError, Error: "expected join_request"})
		_ = conn.Close()
		return
	}

	client, accepted, ok := s.registerClient(conn)
	if !ok {
		_ = encoder.Encode(netcode.Message{Kind: netcode.MessageError, Error: "match already has two players"})
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

func (s *Server) registerClient(conn net.Conn) (*serverClient, netcode.Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	team, ok := s.nextOpenTeamLocked()
	if !ok {
		return nil, netcode.Message{}, false
	}
	clientID := fmt.Sprintf("client-%d", atomic.AddUint64(&s.nextClientID, 1))
	client := &serverClient{id: clientID, team: team, conn: conn}
	s.clients[clientID] = client
	s.teamOwners[team] = clientID
	s.currentInput[team] = sim.InputFrame{ClientID: clientID, Team: team}
	snapshot := cloneGameState(s.state)
	accepted := netcode.Message{
		Kind:     netcode.MessageJoinAccepted,
		MatchID:  s.matchID,
		ClientID: clientID,
		Team:     team,
		TickRate: sim.TickRate,
		State:    &snapshot,
	}
	return client, accepted, true
}

func (s *Server) nextOpenTeamLocked() (sim.Team, bool) {
	if _, ok := s.teamOwners[sim.TeamHome]; !ok {
		return sim.TeamHome, true
	}
	if _, ok := s.teamOwners[sim.TeamAway]; !ok {
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
	if owner, ok := s.teamOwners[client.team]; ok && owner == clientID {
		delete(s.teamOwners, client.team)
		delete(s.currentInput, client.team)
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
		s.currentInput[client.team] = input
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
			snapshot, clients := s.stepAndSnapshot()
			message := netcode.Message{
				Kind:    netcode.MessageSnapshot,
				MatchID: s.matchID,
				Tick:    snapshot.Tick,
				State:   &snapshot,
			}
			for _, client := range clients {
				if err := client.send(message); err != nil {
					s.unregisterClient(client.id)
				}
			}
		}
	}
}

func (s *Server) stepAndSnapshot() (sim.GameState, []*serverClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputs := make([]sim.InputFrame, 0, 2)
	for _, team := range []sim.Team{sim.TeamHome, sim.TeamAway} {
		clientID, ok := s.teamOwners[team]
		if !ok {
			continue
		}
		input := s.currentInput[team]
		input.ClientID = clientID
		input.Team = team
		input.Tick = s.state.Tick + 1
		inputs = append(inputs, input)
	}

	sim.Step(&s.state, inputs)
	snapshot := cloneGameState(s.state)
	clients := make([]*serverClient, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	return snapshot, clients
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
