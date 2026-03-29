package netcode

import (
	"bufio"
	"encoding/json"
	"errors"
	"hockeyv2/internal/sim"
	"net"
	"testing"
	"time"
)

func TestMessageFromInputRoundTrip(t *testing.T) {
	input := sim.InputFrame{
		ClientID:  "client-1",
		Team:      sim.TeamAway,
		Tick:      42,
		Move:      sim.Vec2{X: 1, Y: -2},
		Shoot:     true,
		Pass:      true,
		Switch:    true,
		Ready:     true,
		ColorPrev: true,
		ColorNext: true,
	}

	message := MessageFromInput(input, "client-override")
	if message.Kind != MessageInputFrame {
		t.Fatalf("expected input frame kind, got %q", message.Kind)
	}
	if message.ClientID != "client-override" {
		t.Fatalf("expected overridden client id, got %q", message.ClientID)
	}

	roundTrip := message.ToInputFrame()
	if roundTrip.ClientID != "client-override" || roundTrip.Team != input.Team || roundTrip.Tick != input.Tick {
		t.Fatalf("unexpected round-trip input frame: %+v", roundTrip)
	}
	if !roundTrip.Shoot || !roundTrip.Pass || !roundTrip.Switch || !roundTrip.Ready || !roundTrip.ColorPrev || !roundTrip.ColorNext {
		t.Fatalf("expected boolean fields to survive round trip, got %+v", roundTrip)
	}
}

func startTestTCPServer(t *testing.T, handler func(net.Conn)) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		handler(conn)
	}()
	return listener.Addr().String()
}

func waitSnapshot(t *testing.T, snapshots <-chan sim.GameState) sim.GameState {
	t.Helper()
	select {
	case snapshot := <-snapshots:
		return snapshot
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for snapshot")
		return sim.GameState{}
	}
}

func waitError(t *testing.T, errs <-chan error) error {
	t.Helper()
	select {
	case err := <-errs:
		return err
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for error")
		return nil
	}
}

func TestDialAcceptsJoinAndStreamsSnapshots(t *testing.T) {
	initial := sim.NewMultiplayerGameState()
	initial.HomeColor = sim.TeamColorGreen
	next := initial
	next.Tick = 7
	next.AwayColor = sim.TeamColorOrange

	addr := startTestTCPServer(t, func(conn net.Conn) {
		defer conn.Close()
		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var join Message
		if err := decoder.Decode(&join); err != nil {
			t.Errorf("decode join: %v", err)
			return
		}
		if join.Kind != MessageJoinRequest {
			t.Errorf("expected join request, got %q", join.Kind)
			return
		}
		if join.RoomCode != "AB12C" || join.RoomName != "Friday Night" || !join.CreateRoom {
			t.Errorf("unexpected join metadata %+v", join)
			return
		}
		if err := encoder.Encode(Message{Kind: MessageJoinAccepted, MatchID: "room-AB12C", RoomCode: "AB12C", RoomName: "Friday Night", ClientID: "client-1", Team: sim.TeamAway, Host: true, State: &initial}); err != nil {
			t.Errorf("encode join accepted: %v", err)
			return
		}
		if err := encoder.Encode(Message{Kind: MessageSnapshot, RoomCode: "AB12C", State: &next}); err != nil {
			t.Errorf("encode snapshot: %v", err)
		}
	})

	client, err := DialRoom(addr, "AB12C", true, "Friday Night")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	if client.Team() != sim.TeamAway {
		t.Fatalf("expected away team, got %q", client.Team())
	}
	if client.ClientID() != "client-1" || client.MatchID() != "room-AB12C" {
		t.Fatalf("unexpected handshake ids: client=%q match=%q", client.ClientID(), client.MatchID())
	}
	if client.RoomCode() != "AB12C" || client.RoomName() != "Friday Night" {
		t.Fatalf("unexpected room metadata code=%q name=%q", client.RoomCode(), client.RoomName())
	}

	first := waitSnapshot(t, client.Snapshots())
	if first.HomeColor != sim.TeamColorGreen {
		t.Fatalf("expected initial snapshot home color green, got %q", first.HomeColor)
	}
	second := waitSnapshot(t, client.Snapshots())
	if second.Tick != 7 || second.AwayColor != sim.TeamColorOrange {
		t.Fatalf("unexpected streamed snapshot: %+v", second)
	}
}

func TestDialReturnsServerError(t *testing.T) {
	addr := startTestTCPServer(t, func(conn net.Conn) {
		defer conn.Close()
		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)
		var join Message
		_ = decoder.Decode(&join)
		_ = encoder.Encode(Message{Kind: MessageError, Error: "room full"})
	})

	client, err := Dial(addr)
	if err == nil {
		client.Close()
		t.Fatalf("expected server error")
	}
	if err.Error() != "room full" {
		t.Fatalf("expected room full error, got %v", err)
	}
}

func TestDialRejectsUnexpectedFirstMessage(t *testing.T) {
	addr := startTestTCPServer(t, func(conn net.Conn) {
		defer conn.Close()
		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)
		var join Message
		_ = decoder.Decode(&join)
		_ = encoder.Encode(Message{Kind: MessagePing})
	})

	client, err := Dial(addr)
	if err == nil {
		client.Close()
		t.Fatalf("expected unexpected first message error")
	}
}

func TestClientSendInputUsesAssignedIdentity(t *testing.T) {
	received := make(chan Message, 1)
	addr := startTestTCPServer(t, func(conn net.Conn) {
		defer conn.Close()
		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)
		var join Message
		if err := decoder.Decode(&join); err != nil {
			t.Errorf("decode join: %v", err)
			return
		}
		if err := encoder.Encode(Message{Kind: MessageJoinAccepted, MatchID: "room-ZXCVB", RoomCode: "ZXCVB", ClientID: "client-2", Team: sim.TeamAway, Host: false}); err != nil {
			t.Errorf("encode accepted: %v", err)
			return
		}
		var input Message
		if err := decoder.Decode(&input); err != nil {
			t.Errorf("decode input: %v", err)
			return
		}
		received <- input
	})

	client, err := DialRoom(addr, "ZXCVB", false, "")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	if err := client.SendInput(sim.InputFrame{Team: sim.TeamHome, ClientID: "wrong", Tick: 9, Pass: true}); err != nil {
		t.Fatalf("send input: %v", err)
	}

	select {
	case message := <-received:
		if message.Kind != MessageInputFrame {
			t.Fatalf("expected input frame message, got %q", message.Kind)
		}
		if message.Team != sim.TeamAway || message.ClientID != "client-2" || message.MatchID != "room-ZXCVB" || message.RoomCode != "ZXCVB" {
			t.Fatalf("unexpected message identity: %+v", message)
		}
		if !message.Pass || message.Tick != 9 {
			t.Fatalf("expected sent payload, got %+v", message)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for encoded input")
	}
}

func TestClientReadLoopReportsServerError(t *testing.T) {
	reader, writer := net.Pipe()
	defer writer.Close()
	client := &Client{
		conn:      reader,
		snapshots: make(chan sim.GameState, 1),
		errs:      make(chan error, 1),
		closed:    make(chan struct{}),
	}
	defer client.Close()

	go client.readLoop(json.NewDecoder(bufio.NewReader(reader)))
	if err := json.NewEncoder(writer).Encode(Message{Kind: MessageError, Error: "boom"}); err != nil {
		t.Fatalf("encode error message: %v", err)
	}

	if err := waitError(t, client.Errors()); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestDeliverSnapshotReplacesOldestWhenChannelIsFull(t *testing.T) {
	ch := make(chan sim.GameState, 1)
	first := sim.GameState{Tick: 1}
	second := sim.GameState{Tick: 2}
	deliverSnapshot(ch, first)
	deliverSnapshot(ch, second)

	got := <-ch
	if got.Tick != 2 {
		t.Fatalf("expected latest snapshot tick 2, got %d", got.Tick)
	}
}

func TestCloneGameStateCopiesSlices(t *testing.T) {
	state := sim.NewGameState()
	copyState := cloneGameState(state)
	copyState.HomeSkaters[0].ID = "changed-home"
	copyState.AwaySkaters[0].ID = "changed-away"

	if state.HomeSkaters[0].ID == "changed-home" || state.AwaySkaters[0].ID == "changed-away" {
		t.Fatalf("expected clone to deep copy skater slices")
	}
}

func TestClientCloseIsIdempotent(t *testing.T) {
	reader, writer := net.Pipe()
	defer writer.Close()
	client := &Client{
		conn:      reader,
		snapshots: make(chan sim.GameState, 1),
		errs:      make(chan error, 1),
		closed:    make(chan struct{}),
	}

	if err := client.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestClientReadLoopStopsAfterClose(t *testing.T) {
	reader, writer := net.Pipe()
	client := &Client{
		conn:      reader,
		snapshots: make(chan sim.GameState, 1),
		errs:      make(chan error, 1),
		closed:    make(chan struct{}),
	}
	go client.readLoop(json.NewDecoder(bufio.NewReader(reader)))
	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	select {
	case err := <-client.Errors():
		if !errors.Is(err, net.ErrClosed) && err != nil {
			t.Fatalf("expected no surfaced read error after close, got %v", err)
		}
	case <-time.After(50 * time.Millisecond):
	}
}

