package app

import (
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/sim"
)

type appScreen int

type menuOption int

type matchMenuMode int

type matchMenuAction int

type onlineField int

type matchMenuState struct {
	Mode     matchMenuMode
	Selected int
}

type launchSetupState struct {
	Active bool
	Mode   menuOption
	Color  sim.TeamColor
}

type launchMenu struct {
	Selected       menuOption
	Color          sim.TeamColor
	Status         string
	Rooms          []discovery.Room
	RoomCursor     int
	OnlineRoomName string
	OnlineRoomCode string
	OnlineFocus    onlineField
}

const (
	appScreenMenu appScreen = iota
	appScreenJoinBrowser
	appScreenOnlineRooms
	appScreenSolo
	appScreenRemote
)

const (
	menuOptionSolo menuOption = iota
	menuOptionHost
	menuOptionJoin
	menuOptionOnline
)

const (
	matchMenuModeHidden matchMenuMode = iota
	matchMenuModePause
	matchMenuModeIntermission
	matchMenuModePostgame
	matchMenuModeDisconnected
)

const (
	matchMenuActionNone matchMenuAction = iota
	matchMenuActionQuit
	matchMenuActionRoomMenu
)

const (
	onlineFieldRoomName onlineField = iota
	onlineFieldRoomCode
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

func opponentColorForSelection(home sim.TeamColor) sim.TeamColor {
	away := nextLauncherColor(home, 1)
	if away == home {
		return sim.TeamColorRed
	}
	return away
}

func (s *launchSetupState) Open(mode menuOption, color sim.TeamColor) {
	s.Active = true
	s.Mode = mode
	s.Color = color
}

func (s *launchSetupState) Close() {
	s.Active = false
	s.Mode = menuOptionSolo
	s.Color = ""
}

func roomKey(room discovery.Room) string {
	return room.Code + "|" + room.Addr
}
