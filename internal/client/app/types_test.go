package app

import (
	"testing"

	"hockeyv2/internal/discovery"
	"hockeyv2/internal/sim"
)

func TestMatchMenuStateLifecycle(t *testing.T) {
	menu := matchMenuState{}
	if menu.Visible() {
		t.Fatalf("expected hidden menu by default")
	}
	menu.Open(matchMenuModePause)
	if !menu.Visible() || menu.Mode != matchMenuModePause || menu.Selected != 0 {
		t.Fatalf("unexpected menu after open: %+v", menu)
	}
	menu.Close()
	if menu.Visible() || menu.Mode != matchMenuModeHidden || menu.Selected != 0 {
		t.Fatalf("unexpected menu after close: %+v", menu)
	}
}

func TestNextLauncherColorWraps(t *testing.T) {
	if got := nextLauncherColor(sim.TeamColorBlue, 1); got != sim.TeamColorRed {
		t.Fatalf("expected blue -> red, got %q", got)
	}
	if got := nextLauncherColor(sim.TeamColorBlack, -1); got != sim.TeamColorRed {
		t.Fatalf("expected black -> red when wrapping backward, got %q", got)
	}
	if got := nextLauncherColor(sim.TeamColorRed, 1); got != sim.TeamColorBlack {
		t.Fatalf("expected red -> black when wrapping forward, got %q", got)
	}
}

func TestAwayColorForSoloUsesDifferentColor(t *testing.T) {
	for _, color := range launcherColorCycle {
		if got := awayColorForSolo(color); got == color {
			t.Fatalf("expected away color different from %q", color)
		}
	}
}

func TestRoomKeyUsesCodeAndAddress(t *testing.T) {
	room := discovery.Room{Code: "AB12", Addr: "127.0.0.1:4242"}
	if got := roomKey(room); got != "AB12|127.0.0.1:4242" {
		t.Fatalf("unexpected room key %q", got)
	}
}
