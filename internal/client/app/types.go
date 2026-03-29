package app

import (
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/sim"
)

type appScreen int

type menuOption int

type matchMenuMode int

type matchMenuAction int

type matchMenuState struct {
	Mode     matchMenuMode
	Selected int
}

type launchMenu struct {
	Selected   menuOption
	SoloColor  sim.TeamColor
	Status     string
	Rooms      []discovery.Room
	RoomCursor int
}

const (
	appScreenMenu appScreen = iota
	appScreenJoinBrowser
	appScreenSolo
	appScreenRemote
)

const (
	menuOptionSolo menuOption = iota
	menuOptionHost
	menuOptionJoin
)

const (
	matchMenuModeHidden matchMenuMode = iota
	matchMenuModePause
	matchMenuModePostgame
	matchMenuModeDisconnected
)

const (
	matchMenuActionNone matchMenuAction = iota
	matchMenuActionQuit
	matchMenuActionRoomMenu
)

var launcherColorCycle = []sim.TeamColor{
	sim.TeamColorBlack,
	sim.TeamColorOrange,
	sim.TeamColorGreen,
	sim.TeamColorBlue,
	sim.TeamColorRed,
}

func (m matchMenuState) Visible() bool {
	return m.Mode != matchMenuModeHidden
}

func (m *matchMenuState) Open(mode matchMenuMode) {
	m.Mode = mode
	m.Selected = 0
}

func (m *matchMenuState) Close() {
	m.Mode = matchMenuModeHidden
	m.Selected = 0
}

func nextLauncherColor(current sim.TeamColor, delta int) sim.TeamColor {
	currentIndex := 0
	for index, candidate := range launcherColorCycle {
		if candidate == current {
			currentIndex = index
			break
		}
	}
	nextIndex := (currentIndex + delta) % len(launcherColorCycle)
	if nextIndex < 0 {
		nextIndex += len(launcherColorCycle)
	}
	return launcherColorCycle[nextIndex]
}

func awayColorForSolo(home sim.TeamColor) sim.TeamColor {
	away := nextLauncherColor(home, 1)
	if away == home {
		return sim.TeamColorRed
	}
	return away
}

func roomKey(room discovery.Room) string {
	return room.Code + "|" + room.Addr
}
