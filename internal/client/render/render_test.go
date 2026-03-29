package render

import (
	"testing"

	"hockeyv2/internal/discovery"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPaletteAndTeamColorLabels(t *testing.T) {
	cases := []struct {
		color sim.TeamColor
		label string
	}{
		{sim.TeamColorBlack, "Black"},
		{sim.TeamColorOrange, "Orange"},
		{sim.TeamColorGreen, "Green"},
		{sim.TeamColorBlue, "Blue"},
		{sim.TeamColorRed, "Red"},
	}
	for _, tc := range cases {
		palette := paletteForTeamColor(tc.color)
		if palette.Primary.A == 0 || palette.Trim.A == 0 {
			t.Fatalf("expected non-empty palette for %q", tc.color)
		}
		if got := TeamColorLabel(tc.color); got != tc.label {
			t.Fatalf("TeamColorLabel(%q) = %q, want %q", tc.color, got, tc.label)
		}
	}

	state := sim.NewGameState()
	state.HomeColor = sim.TeamColorGreen
	state.AwayColor = sim.TeamColorOrange
	if got := paletteForTeam(state, sim.TeamHome); got != paletteForTeamColor(sim.TeamColorGreen) {
		t.Fatalf("expected home palette to use home color")
	}
	if got := teamColorForDisplay(state, sim.TeamAway); got != sim.TeamColorOrange {
		t.Fatalf("expected away display color orange, got %q", got)
	}
}

func TestLauncherAndReadyOverlayRects(t *testing.T) {
	footer := LauncherFooterRect()
	if footer.W <= 0 || footer.H <= 0 {
		t.Fatalf("unexpected launcher footer rect %+v", footer)
	}
	if prev := LauncherSoloColorPrevRect(); prev.X < footer.X || prev.Y < footer.Y {
		t.Fatalf("expected prev button inside footer, got %+v footer=%+v", prev, footer)
	}
	if next := LauncherSoloColorNextRect(); next.X <= LauncherSoloColorLabelRect().X {
		t.Fatalf("expected next button to appear after label, got %+v", next)
	}

	card0 := MenuOptionRect(0)
	card1 := MenuOptionRect(1)
	if card1.Y <= card0.Y {
		t.Fatalf("expected later menu card lower on screen: %+v vs %+v", card0, card1)
	}

	homeCard := ReadyOverlayCardRect(sim.TeamHome)
	awayCard := ReadyOverlayCardRect(sim.TeamAway)
	if awayCard.X <= homeCard.X {
		t.Fatalf("expected away ready card to be to the right of home card")
	}
	if ReadyOverlayReadyRect(sim.TeamHome).Y <= ReadyOverlayColorLabelRect(sim.TeamHome).Y {
		t.Fatalf("expected ready button below color controls")
	}
}

func TestJoinRoomCardsWindowAndTruncation(t *testing.T) {
	if cards := JoinRoomCards(0, 0); cards != nil {
		t.Fatalf("expected no cards for empty room list, got %+v", cards)
	}
	cards := JoinRoomCards(6, 4)
	if len(cards) != 4 {
		t.Fatalf("expected 4 visible cards, got %d", len(cards))
	}
	if cards[0].Index != 1 || cards[len(cards)-1].Index != 4 {
		t.Fatalf("unexpected scrolling window %+v", cards)
	}
	if got := truncateLabel("Go Hockey LAN Room", 8); got != "Go Ho..." {
		t.Fatalf("unexpected truncated label %q", got)
	}
	if got := truncateLabel("ABC", 3); got != "ABC" {
		t.Fatalf("expected short label unchanged, got %q", got)
	}
}

func TestRenderDrawFunctionsSmoke(t *testing.T) {
	screen := ebiten.NewImage(int(sim.WindowWidth), int(sim.WindowHeight))

	state := sim.NewGameState()
	DrawMatch(screen, state, sim.TeamHome)
	DrawSoloHUD(screen, state, "Solo mode")
	DrawNetworkHUD(screen, state, "Online HOME", "Connected")

	multiplayer := sim.NewMultiplayerGameState()
	DrawReadyOverlay(screen, multiplayer, sim.TeamHome, "Connected to online match")
	multiplayer.Phase = sim.MatchPhaseIntermission
	multiplayer.PhaseTicks = sim.TickRate
	multiplayer.LastIntermissionStats = sim.PeriodStats{Period: 1, Home: sim.TeamPeriodStats{ShotsOnGoal: 5, Goals: 2}, Away: sim.TeamPeriodStats{ShotsOnGoal: 3, Goals: 1}}
	DrawReadyOverlay(screen, multiplayer, sim.TeamAway, "Intermission")

	DrawLauncherMenu(screen, LauncherMenuModel{SelectedOption: 0, SoloColor: sim.TeamColorBlue, Status: "Ready", RoomCount: 2})
	DrawLauncherMenu(screen, LauncherMenuModel{SelectedOption: 1, SoloColor: sim.TeamColorBlue, Status: "Hosting", RoomCount: 0})
	DrawLauncherMenu(screen, LauncherMenuModel{SelectedOption: 2, SoloColor: sim.TeamColorBlue, Status: "Browsing", RoomCount: 3})

	DrawJoinBrowser(screen, JoinBrowserModel{Status: "Searching"})
	DrawJoinBrowser(screen, JoinBrowserModel{
		Rooms: []discovery.Room{
			{Code: "AB12", Name: "Skate Shack", Addr: "192.168.1.10:4242", Status: discovery.Status{Players: 1, Capacity: 2}},
			{Code: "CD34", Name: "Full House", Addr: "192.168.1.11:4242", Status: discovery.Status{Players: 2, Capacity: 2}},
		},
		SelectedRoom: 1,
		Status:       "Connected",
	})
}
