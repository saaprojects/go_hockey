package server

import (
	"bufio"
	"encoding/json"
	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"
	"net"
	"testing"
	"time"
)

func startTestServer(t *testing.T) (*Server, chan error) {
	t.Helper()
	srv, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve()
	}()
	time.Sleep(40 * time.Millisecond)
	return srv, serveDone
}

func stopTestServer(t *testing.T, srv *Server, serveDone chan error) {
	t.Helper()
	_ = srv.Close()
	select {
	case <-serveDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for server shutdown")
	}
}

func TestDefaultRoomFirstTwoClientsGetHomeAndAway(t *testing.T) {
	srv, serveDone := startTestServer(t)
	defer stopTestServer(t, srv, serveDone)

	clientA, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial first client: %v", err)
	}
	defer clientA.Close()

	clientB, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial second client: %v", err)
	}
	defer clientB.Close()

	if clientA.Team() != sim.TeamHome {
		t.Fatalf("expected first client to be home, got %q", clientA.Team())
	}
	if clientB.Team() != sim.TeamAway {
		t.Fatalf("expected second client to be away, got %q", clientB.Team())
	}
}

func TestDefaultRoomThirdClientIsRejectedWhenFull(t *testing.T) {
	srv, serveDone := startTestServer(t)
	defer stopTestServer(t, srv, serveDone)

	clientA, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial first client: %v", err)
	}
	defer clientA.Close()

	clientB, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial second client: %v", err)
	}
	defer clientB.Close()

	clientC, err := netcode.Dial(srv.Addr())
	if err == nil {
		clientC.Close()
		t.Fatalf("expected third client to be rejected")
	}
}

func TestCreateRoomAssignsCodeAndJoinByCodeWorks(t *testing.T) {
	srv, serveDone := startTestServer(t)
	defer stopTestServer(t, srv, serveDone)

	host, err := netcode.DialRoom(srv.Addr(), "", true, "Friday Night")
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	defer host.Close()

	if host.Team() != sim.TeamHome {
		t.Fatalf("expected room creator to be home, got %q", host.Team())
	}
	if len(host.RoomCode()) != onlineCodeLength {
		t.Fatalf("expected generated %d-char room code, got %q", onlineCodeLength, host.RoomCode())
	}
	if host.RoomName() != "Friday Night" {
		t.Fatalf("expected room name to round-trip, got %q", host.RoomName())
	}

	joiner, err := netcode.DialRoom(srv.Addr(), host.RoomCode(), false, "")
	if err != nil {
		t.Fatalf("join room by code: %v", err)
	}
	defer joiner.Close()

	if joiner.Team() != sim.TeamAway {
		t.Fatalf("expected room joiner to be away, got %q", joiner.Team())
	}
	if joiner.RoomCode() != host.RoomCode() {
		t.Fatalf("expected joined room code %q, got %q", host.RoomCode(), joiner.RoomCode())
	}
}

func TestJoinMissingRoomIsRejected(t *testing.T) {
	srv, serveDone := startTestServer(t)
	defer stopTestServer(t, srv, serveDone)

	client, err := netcode.DialRoom(srv.Addr(), "ABCDE", false, "")
	if err == nil {
		client.Close()
		t.Fatalf("expected missing room join to fail")
	}
	if err.Error() != "room not found" {
		t.Fatalf("unexpected missing room error %v", err)
	}
}

func TestDifferentRoomsDoNotShareCapacity(t *testing.T) {
	srv, serveDone := startTestServer(t)
	defer stopTestServer(t, srv, serveDone)

	roomAHost, err := netcode.DialRoom(srv.Addr(), "", true, "Room A")
	if err != nil {
		t.Fatalf("create room A: %v", err)
	}
	defer roomAHost.Close()

	roomBHost, err := netcode.DialRoom(srv.Addr(), "", true, "Room B")
	if err != nil {
		t.Fatalf("create room B: %v", err)
	}
	defer roomBHost.Close()

	roomAJoiner, err := netcode.DialRoom(srv.Addr(), roomAHost.RoomCode(), false, "")
	if err != nil {
		t.Fatalf("join room A: %v", err)
	}
	defer roomAJoiner.Close()

	thirdInA, err := netcode.DialRoom(srv.Addr(), roomAHost.RoomCode(), false, "")
	if err == nil {
		thirdInA.Close()
		t.Fatalf("expected room A to be full")
	}

	roomBJoiner, err := netcode.DialRoom(srv.Addr(), roomBHost.RoomCode(), false, "")
	if err != nil {
		t.Fatalf("join room B: %v", err)
	}
	defer roomBJoiner.Close()
}

func TestNormalizeRoomCode(t *testing.T) {
	if got := normalizeRoomCode(" ab-12cdef "); got != "AB2CD" {
		t.Fatalf("unexpected normalized room code %q", got)
	}
	if got := normalizeRoomCode("   "); got != defaultRoomCode {
		t.Fatalf("expected blank room code to normalize to default room, got %q", got)
	}
}

func TestSetLobbyColorsUpdatesDefaultRoomState(t *testing.T) {
	srv, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()

	srv.SetLobbyColors(sim.TeamColorOrange, sim.TeamColorBlue)
	room := srv.ensureRoomLocked(defaultRoomCode, "")
	if room.state.HomeColor != sim.TeamColorOrange || room.state.AwayColor != sim.TeamColorBlue {
		t.Fatalf("unexpected lobby colors home=%q away=%q", room.state.HomeColor, room.state.AwayColor)
	}
}

func TestServerCloneGameStateCopiesSlices(t *testing.T) {
	state := sim.NewGameState()
	copyState := cloneGameState(state)
	copyState.HomeSkaters[0].ID = "changed-home"
	copyState.AwaySkaters[0].ID = "changed-away"
	if state.HomeSkaters[0].ID == "changed-home" || state.AwaySkaters[0].ID == "changed-away" {
		t.Fatalf("expected server clone to deep copy skater slices")
	}
}

func TestStepAndSnapshotAdvancesEachRoom(t *testing.T) {
	defaultState := sim.NewMultiplayerGameState()
	customState := sim.NewMultiplayerGameState()
	srv := &Server{
		rooms: map[string]*matchRoom{
			defaultRoomCode: {
				code:         defaultRoomCode,
				state:        defaultState,
				teamOwners:   map[sim.Team]string{sim.TeamHome: "client-1"},
				currentInput: map[sim.Team]sim.InputFrame{sim.TeamHome: {Move: sim.Vec2{X: 1}}},
			},
			"ABCDE": {
				code:         "ABCDE",
				name:         "Friday Night",
				state:        customState,
				teamOwners:   map[sim.Team]string{sim.TeamHome: "client-2"},
				currentInput: map[sim.Team]sim.InputFrame{sim.TeamHome: {Move: sim.Vec2{X: -1}}},
			},
		},
		clients: map[string]*serverClient{
			"client-1": {id: "client-1"},
			"client-2": {id: "client-2"},
		},
	}

	fanouts := srv.stepAndSnapshot()
	if len(fanouts) != 2 {
		t.Fatalf("expected fanout for both rooms, got %d", len(fanouts))
	}
	seen := map[string]bool{}
	for _, fanout := range fanouts {
		seen[fanout.roomCode] = true
		if fanout.snapshot.Tick != 1 {
			t.Fatalf("expected stepped tick 1 for room %q, got %d", fanout.roomCode, fanout.snapshot.Tick)
		}
		if len(fanout.clients) != 1 {
			t.Fatalf("expected one client for room %q, got %d", fanout.roomCode, len(fanout.clients))
		}
	}
	if !seen[defaultRoomCode] || !seen["ABCDE"] {
		t.Fatalf("expected snapshots for default and custom rooms, got %+v", seen)
	}
}

func TestServerClientSendEncodesMessage(t *testing.T) {
	reader, writer := net.Pipe()
	defer reader.Close()
	defer writer.Close()

	client := &serverClient{id: "client-1", encoder: json.NewEncoder(writer)}
	message := netcode.Message{Kind: netcode.MessagePing, ClientID: "client-1"}

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.send(message)
	}()

	var decoded netcode.Message
	if err := json.NewDecoder(bufio.NewReader(reader)).Decode(&decoded); err != nil {
		t.Fatalf("decode sent message: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("send message: %v", err)
	}
	if decoded.Kind != netcode.MessagePing || decoded.ClientID != "client-1" {
		t.Fatalf("unexpected decoded message %+v", decoded)
	}
}

func TestListRoomsReturnsCustomRoomsAndExcludesDefaultRoom(t *testing.T) {
	srv, serveDone := startTestServer(t)
	defer stopTestServer(t, srv, serveDone)

	srv.SetLobbyColors(sim.TeamColorOrange, sim.TeamColorBlue)

	openHost, err := netcode.DialRoom(srv.Addr(), "", true, "Open Room")
	if err != nil {
		t.Fatalf("create open room: %v", err)
	}
	defer openHost.Close()

	fullHost, err := netcode.DialRoom(srv.Addr(), "", true, "Full Room")
	if err != nil {
		t.Fatalf("create full room: %v", err)
	}
	defer fullHost.Close()

	fullJoiner, err := netcode.DialRoom(srv.Addr(), fullHost.RoomCode(), false, "")
	if err != nil {
		t.Fatalf("fill room: %v", err)
	}
	defer fullJoiner.Close()

	rooms, err := netcode.ListRooms(srv.Addr())
	if err != nil {
		t.Fatalf("list rooms: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("expected 2 custom rooms, got %+v", rooms)
	}
	if rooms[0].Name != "Open Room" || rooms[0].Code != openHost.RoomCode() {
		t.Fatalf("expected joinable room first, got %+v", rooms[0])
	}
	if !rooms[0].Joinable() {
		t.Fatalf("expected first room to be joinable, got %+v", rooms[0])
	}
	if rooms[1].Name != "Full Room" || rooms[1].Code != fullHost.RoomCode() {
		t.Fatalf("expected full room second, got %+v", rooms[1])
	}
	if rooms[1].Joinable() {
		t.Fatalf("expected second room to be full, got %+v", rooms[1])
	}
}
