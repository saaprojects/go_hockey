package discovery

import (
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
