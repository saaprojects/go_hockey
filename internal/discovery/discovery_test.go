package discovery

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBrowserDiscoversAdvertisedRoom(t *testing.T) {
	advertiser, err := NewAdvertiserWithConfig(AdvertiserConfig{
		ListenAddr: "127.0.0.1:0",
		TCPAddr:    "127.0.0.1:4242",
		RoomName:   "Test Host",
		RoomCode:   "AB12",
		StatusFunc: func() Status {
			return Status{Players: 1, Capacity: 2}
		},
	})
	if err != nil {
		t.Fatalf("start advertiser: %v", err)
	}
	defer advertiser.Close()

	browser, err := NewBrowserWithConfig(BrowserConfig{
		ListenAddr:    "127.0.0.1:0",
		ProbeTargets:  []string{advertiser.Addr()},
		ProbeInterval: 20 * time.Millisecond,
		EntryTTL:      120 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("start browser: %v", err)
	}
	defer browser.Close()

	rooms := waitForRooms(t, browser.Updates(), 400*time.Millisecond, func(rooms []Room) bool {
		return len(rooms) == 1
	})

	if got := rooms[0].Code; got != "AB12" {
		t.Fatalf("expected room code AB12, got %q", got)
	}
	if got := rooms[0].Name; got != "Test Host" {
		t.Fatalf("expected room name Test Host, got %q", got)
	}
	if got := rooms[0].Addr; got != "127.0.0.1:4242" {
		t.Fatalf("expected room addr 127.0.0.1:4242, got %q", got)
	}
	if got := rooms[0].Status.Players; got != 1 {
		t.Fatalf("expected 1 player, got %d", got)
	}
	if !rooms[0].Joinable() {
		t.Fatalf("expected discovered room to be joinable")
	}
}

func TestBrowserRemovesRoomAfterAdvertiserStops(t *testing.T) {
	advertiser, err := NewAdvertiserWithConfig(AdvertiserConfig{
		ListenAddr: "127.0.0.1:0",
		TCPAddr:    "127.0.0.1:4242",
		RoomName:   "Stale Host",
		RoomCode:   "CD34",
	})
	if err != nil {
		t.Fatalf("start advertiser: %v", err)
	}

	browser, err := NewBrowserWithConfig(BrowserConfig{
		ListenAddr:    "127.0.0.1:0",
		ProbeTargets:  []string{advertiser.Addr()},
		ProbeInterval: 20 * time.Millisecond,
		EntryTTL:      80 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("start browser: %v", err)
	}
	defer browser.Close()

	waitForRooms(t, browser.Updates(), 400*time.Millisecond, func(rooms []Room) bool {
		return len(rooms) == 1
	})

	if err := advertiser.Close(); err != nil {
		t.Fatalf("close advertiser: %v", err)
	}

	rooms := waitForRooms(t, browser.Updates(), 500*time.Millisecond, func(rooms []Room) bool {
		return len(rooms) == 0
	})
	if len(rooms) != 0 {
		t.Fatalf("expected no rooms after advertiser stopped, got %d", len(rooms))
	}
}

func waitForRooms(t *testing.T, updates <-chan []Room, timeout time.Duration, predicate func([]Room) bool) []Room {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case rooms, ok := <-updates:
			if !ok {
				t.Fatalf("updates channel closed before condition was met")
			}
			if predicate(rooms) {
				return rooms
			}
		case <-deadline:
			t.Fatalf("timed out waiting for room update")
		}
	}
}

func TestRoomJoinable(t *testing.T) {
	if !(Room{Status: Status{Players: 1, Capacity: 2}}).Joinable() {
		t.Fatalf("expected partially full room to be joinable")
	}
	if (Room{Status: Status{Players: 2, Capacity: 2}}).Joinable() {
		t.Fatalf("expected full room to be unjoinable")
	}
	if !(Room{Status: Status{Players: 5, Capacity: 0}}).Joinable() {
		t.Fatalf("expected open capacity room to be joinable")
	}
}

func TestNormalizeStatus(t *testing.T) {
	if got := normalizeStatus(Status{Players: -1, Capacity: 0}); got.Players != 0 || got.Capacity != 2 {
		t.Fatalf("unexpected normalized status %+v", got)
	}
	if got := normalizeStatus(Status{Players: 5, Capacity: 2}); got.Players != 2 || got.Capacity != 2 {
		t.Fatalf("unexpected capped status %+v", got)
	}
}

func TestRoomKey(t *testing.T) {
	if got := roomKey(Room{Code: "AB12", Addr: "127.0.0.1:4242"}); got != "AB12|127.0.0.1:4242" {
		t.Fatalf("unexpected room key %q", got)
	}
}

func TestIsUsableAdvertisedHost(t *testing.T) {
	cases := map[string]bool{
		"":            false,
		"localhost":   false,
		"0.0.0.0":     false,
		"127.0.0.1":   false,
		"192.168.1.4": true,
		"my-host":     true,
	}
	for host, want := range cases {
		if got := isUsableAdvertisedHost(host); got != want {
			t.Fatalf("isUsableAdvertisedHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestBroadcastTargetsIncludeLimitedBroadcast(t *testing.T) {
	targets := broadcastTargets(4242)
	found := false
	for _, target := range targets {
		if target.String() == "255.255.255.255:4242" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected limited broadcast target in %+v", targets)
	}
}

func TestDefaultRoomNameIsNonEmpty(t *testing.T) {
	if got := strings.TrimSpace(defaultRoomName()); got == "" {
		t.Fatalf("expected non-empty default room name")
	}
}

func TestRandomRoomCode(t *testing.T) {
	code, err := randomRoomCode(bytes.NewReader([]byte{0, 1, 2, 3}))
	if err != nil {
		t.Fatalf("randomRoomCode: %v", err)
	}
	if len(code) != roomCodeLength {
		t.Fatalf("expected %d-character code, got %q", roomCodeLength, code)
	}
	for _, ch := range code {
		if !strings.ContainsRune(roomCodeAlphabet, ch) {
			t.Fatalf("unexpected room code character %q in %q", ch, code)
		}
	}
}

func TestRandomRoomCodeReturnsReadError(t *testing.T) {
	if _, err := randomRoomCode(bytes.NewReader([]byte{1, 2})); err == nil {
		t.Fatalf("expected short read error")
	}
}

func TestMustResolveUDPAddrPanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic for invalid address")
		}
	}()
	_ = mustResolveUDPAddr("not an addr with spaces")
}

func TestIsClosedNetworkError(t *testing.T) {
	if !isClosedNetworkError(errors.New("use of closed network connection")) {
		t.Fatalf("expected closed network error to be detected")
	}
	if isClosedNetworkError(errors.New("different error")) {
		t.Fatalf("did not expect unrelated error to match")
	}
}
