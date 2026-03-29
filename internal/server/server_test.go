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

func TestFirstTwoClientsGetHomeAndAway(t *testing.T) {
	srv, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve()
	}()
	defer func() {
		_ = srv.Close()
		select {
		case <-serveDone:
		case <-time.After(500 * time.Millisecond):
		}
	}()

	time.Sleep(40 * time.Millisecond)

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

func TestThirdClientIsRejectedWhenMatchIsFull(t *testing.T) {
	srv, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve()
	}()
	defer func() {
		_ = srv.Close()
		select {
		case <-serveDone:
		case <-time.After(500 * time.Millisecond):
		}
	}()

	time.Sleep(40 * time.Millisecond)

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

func TestNextOpenTeamLocked(t *testing.T) {
	srv := &Server{teamOwners: map[sim.Team]string{}}
	if team, ok := srv.nextOpenTeamLocked(); !ok || team != sim.TeamHome {
		t.Fatalf("expected home to open first, got team=%q ok=%v", team, ok)
	}
	srv.teamOwners[sim.TeamHome] = "client-1"
	if team, ok := srv.nextOpenTeamLocked(); !ok || team != sim.TeamAway {
		t.Fatalf("expected away to open second, got team=%q ok=%v", team, ok)
	}
	srv.teamOwners[sim.TeamAway] = "client-2"
	if team, ok := srv.nextOpenTeamLocked(); ok || team != sim.TeamNone {
		t.Fatalf("expected no open teams, got team=%q ok=%v", team, ok)
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

func TestStepAndSnapshotUsesOwnedTeamInputs(t *testing.T) {
	srv := &Server{
		state:        sim.NewMultiplayerGameState(),
		clients:      map[string]*serverClient{"client-1": {id: "client-1"}, "client-2": {id: "client-2"}},
		teamOwners:   map[sim.Team]string{sim.TeamHome: "client-1", sim.TeamAway: "client-2"},
		currentInput: map[sim.Team]sim.InputFrame{sim.TeamHome: {Move: sim.Vec2{X: 1}}, sim.TeamAway: {Move: sim.Vec2{X: -1}}},
	}
	startTick := srv.state.Tick
	snapshot, clients := srv.stepAndSnapshot()
	if snapshot.Tick != startTick+1 {
		t.Fatalf("expected snapshot tick %d, got %d", startTick+1, snapshot.Tick)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients in snapshot fanout, got %d", len(clients))
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
