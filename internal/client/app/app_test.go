package app

import (
	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"
	"testing"
)

func TestLocalJoinAddressNormalizesWildcardHosts(t *testing.T) {
	cases := map[string]string{
		":4242":            "127.0.0.1:4242",
		"0.0.0.0:4242":     "127.0.0.1:4242",
		"[::]:4242":        "127.0.0.1:4242",
		"192.168.1.4:4242": "192.168.1.4:4242",
	}
	for input, want := range cases {
		if got := localJoinAddress(input); got != want {
			t.Fatalf("localJoinAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoteWindowTitleUppercasesTeam(t *testing.T) {
	if got := remoteWindowTitle("home"); got != "Go Hockey - Online HOME" {
		t.Fatalf("unexpected title %q", got)
	}
}

func TestOnlineHelpers(t *testing.T) {
	if got := onlineServerAddress(); got != defaultOnlineServerAddr {
		t.Fatalf("unexpected default online server address %q", got)
	}
	t.Setenv(onlineServerEnvVar, "play.example.com:4242")
	if got := onlineServerAddress(); got != "play.example.com:4242" {
		t.Fatalf("unexpected overridden online server address %q", got)
	}
	if normalizedOnlineRoomName("  Friday Night  ") != "Friday Night" {
		t.Fatalf("expected trimmed room name")
	}
	if normalizedOnlineRoomName("   ") == "" {
		t.Fatalf("expected fallback room name when blank")
	}
	if status := onlineConnectionErrorStatus(nil); status != "" {
		t.Fatalf("expected blank status for nil error, got %q", status)
	}
}

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

func TestLaunchSetupStateLifecycle(t *testing.T) {
	setup := launchSetupState{}
	setup.Open(menuOptionHost, sim.TeamColorOrange)
	if !setup.Active || setup.Mode != menuOptionHost || setup.Color != sim.TeamColorOrange {
		t.Fatalf("unexpected setup state after open: %+v", setup)
	}
	setup.Close()
	if setup.Active || setup.Mode != menuOptionSolo || setup.Color != "" {
		t.Fatalf("unexpected setup state after close: %+v", setup)
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

func TestOpponentColorForSelectionUsesDifferentColor(t *testing.T) {
	for _, color := range launcherColorCycle {
		if got := opponentColorForSelection(color); got == color {
			t.Fatalf("expected opponent color different from %q", color)
		}
	}
}

func TestRoomKeyUsesCodeAndAddress(t *testing.T) {
	room := discovery.Room{Code: "AB12", Addr: "127.0.0.1:4242"}
	if got := roomKey(room); got != "AB12|127.0.0.1:4242" {
		t.Fatalf("unexpected room key %q", got)
	}
}

func TestSetDiscoveredRoomsPreservesSelectedRoom(t *testing.T) {
	app := &App{menu: launchMenu{Rooms: []discovery.Room{{Code: "AB12", Addr: "1.1.1.1:4242"}, {Code: "CD34", Addr: "2.2.2.2:4242"}}, RoomCursor: 1}}
	app.setDiscoveredRooms([]discovery.Room{{Code: "ZZ99", Addr: "3.3.3.3:4242"}, {Code: "CD34", Addr: "2.2.2.2:4242"}})
	if app.menu.RoomCursor != 1 {
		t.Fatalf("expected cursor to follow matching room, got %d", app.menu.RoomCursor)
	}
}

func TestSetDiscoveredRoomsClampsCursor(t *testing.T) {
	app := &App{menu: launchMenu{RoomCursor: 5}}
	app.setDiscoveredRooms([]discovery.Room{{Code: "AB12", Addr: "1.1.1.1:4242"}})
	if app.menu.RoomCursor != 0 {
		t.Fatalf("expected cursor clamped to 0, got %d", app.menu.RoomCursor)
	}

	app.menu.RoomCursor = -1
	app.setDiscoveredRooms([]discovery.Room{{Code: "AB12", Addr: "1.1.1.1:4242"}, {Code: "CD34", Addr: "2.2.2.2:4242"}})
	if app.menu.RoomCursor != 0 {
		t.Fatalf("expected negative cursor clamped to 0, got %d", app.menu.RoomCursor)
	}
}

func TestJoinRoomStatusGuards(t *testing.T) {
	app := &App{}
	if err := app.joinRoom(0); err != nil {
		t.Fatalf("join room without browser: %v", err)
	}
	if app.menu.Status != "LAN discovery unavailable" {
		t.Fatalf("unexpected status %q", app.menu.Status)
	}

	app.browser = &discovery.Browser{}
	if err := app.joinRoom(0); err != nil {
		t.Fatalf("join room without rooms: %v", err)
	}
	if app.menu.Status != "Searching for LAN rooms" {
		t.Fatalf("unexpected status %q", app.menu.Status)
	}

	app.menu.Rooms = []discovery.Room{{Code: "AB12", Addr: "1.1.1.1:4242", Status: discovery.Status{Players: 2, Capacity: 2}}}
	if err := app.joinRoom(0); err != nil {
		t.Fatalf("join full room: %v", err)
	}
	if app.menu.Status != "That room is already full" {
		t.Fatalf("unexpected status %q", app.menu.Status)
	}
}

func TestJoinOnlineRoomGuards(t *testing.T) {
	app := &App{menu: launchMenu{OnlineRoomCode: "", OnlineFocus: onlineFieldRoomCode}}
	if err := app.joinOnlineRoomByCode(); err != nil {
		t.Fatalf("join online room without code: %v", err)
	}
	if app.menu.Status != "Enter a 5-character room code" {
		t.Fatalf("unexpected join status %q", app.menu.Status)
	}

	app.menu.OnlineRoomCode = "AB"
	if err := app.joinOnlineRoomByCode(); err != nil {
		t.Fatalf("join online room with short code: %v", err)
	}
	if app.menu.Status != "Room codes are 5 characters" {
		t.Fatalf("unexpected short code status %q", app.menu.Status)
	}
}

func TestReturnToMenuResetsLauncherState(t *testing.T) {
	app := &App{screen: appScreenRemote, remote: &RemoteGame{}, solo: NewSoloGame(), setup: launchSetupState{Active: true, Mode: menuOptionHost, Color: sim.TeamColorRed}}
	app.returnToMenu("Ready")
	if app.screen != appScreenMenu || app.remote != nil || app.solo != nil || app.setup.Active {
		t.Fatalf("expected return to launcher state, got screen=%v remote=%v solo=%v setup=%+v", app.screen, app.remote, app.solo, app.setup)
	}
	if app.menu.Status != "Ready" {
		t.Fatalf("unexpected status %q", app.menu.Status)
	}
}

func TestReturnToRoomMenuUsesRecordedRoomScreen(t *testing.T) {
	app := &App{screen: appScreenRemote, remote: &RemoteGame{}, roomMenuScreen: appScreenOnlineRooms}
	app.returnToRoomMenu("Created room AB12C")
	if app.screen != appScreenOnlineRooms || app.menu.Status != "Created room AB12C" {
		t.Fatalf("expected online rooms state, got screen=%v status=%q", app.screen, app.menu.Status)
	}

	app = &App{screen: appScreenRemote, remote: &RemoteGame{}, browser: &discovery.Browser{}, roomMenuScreen: appScreenJoinBrowser}
	app.returnToRoomMenu("")
	if app.screen != appScreenJoinBrowser || app.menu.Status != "Searching for LAN rooms" {
		t.Fatalf("expected join browser search state, got screen=%v status=%q", app.screen, app.menu.Status)
	}

	app = &App{screen: appScreenRemote, remote: &RemoteGame{}, roomMenuScreen: appScreenMenu}
	app.returnToRoomMenu("")
	if app.screen != appScreenMenu || app.menu.Status != "Back at launcher" {
		t.Fatalf("expected launcher fallback, got screen=%v status=%q", app.screen, app.menu.Status)
	}
}

func TestActivateMenuOptionTransitions(t *testing.T) {
	app := &App{menu: launchMenu{Color: sim.TeamColorGreen}}
	if err := app.activateMenuOption(menuOptionSolo); err != nil {
		t.Fatalf("activate solo: %v", err)
	}
	if app.screen != appScreenMenu || app.solo != nil || !app.setup.Active || app.setup.Mode != menuOptionSolo {
		t.Fatalf("expected solo setup modal, got screen=%v solo=%v setup=%+v", app.screen, app.solo, app.setup)
	}
	if app.setup.Color != sim.TeamColorGreen {
		t.Fatalf("expected setup color to mirror launcher color, got %q", app.setup.Color)
	}
	if err := app.confirmLaunchSetup(); err != nil {
		t.Fatalf("confirm solo setup: %v", err)
	}
	if app.screen != appScreenSolo || app.solo == nil || app.setup.Active {
		t.Fatalf("expected solo game to start, got screen=%v solo=%v setup=%+v", app.screen, app.solo, app.setup)
	}
	if app.solo.state.HomeColor != sim.TeamColorGreen || app.solo.state.AwayColor == sim.TeamColorGreen {
		t.Fatalf("unexpected solo game colors: home=%q away=%q", app.solo.state.HomeColor, app.solo.state.AwayColor)
	}

	app = &App{menu: launchMenu{Color: sim.TeamColorOrange}}
	if err := app.activateMenuOption(menuOptionHost); err != nil {
		t.Fatalf("activate host: %v", err)
	}
	if app.screen != appScreenMenu || !app.setup.Active || app.setup.Mode != menuOptionHost || app.setup.Color != sim.TeamColorOrange {
		t.Fatalf("expected host setup modal, got screen=%v setup=%+v", app.screen, app.setup)
	}

	app = &App{}
	if err := app.activateMenuOption(menuOptionJoin); err != nil {
		t.Fatalf("activate join without browser: %v", err)
	}
	if app.screen != appScreenJoinBrowser || app.menu.Status != "LAN discovery unavailable" {
		t.Fatalf("unexpected join browser state screen=%v status=%q", app.screen, app.menu.Status)
	}

	app = &App{browser: &discovery.Browser{}}
	if err := app.activateMenuOption(menuOptionJoin); err != nil {
		t.Fatalf("activate join with browser: %v", err)
	}
	if app.menu.Status != "Searching for LAN rooms" {
		t.Fatalf("unexpected join search status %q", app.menu.Status)
	}

	previousListOnlineRooms := listOnlineRooms
	listOnlineRooms = func(addr string) ([]netcode.RoomSummary, error) {
		return nil, nil
	}
	defer func() { listOnlineRooms = previousListOnlineRooms }()

	app = &App{}
	if err := app.activateMenuOption(menuOptionOnline); err != nil {
		t.Fatalf("activate online: %v", err)
	}
	if app.screen != appScreenOnlineRooms || app.menu.Status != onlineRoomListLoadingStatus || app.menu.OnlineFocus != onlineFieldRoomName {
		t.Fatalf("unexpected online room state screen=%v status=%q focus=%v", app.screen, app.menu.Status, app.menu.OnlineFocus)
	}
}

func TestRemoteConsumeActionClearsAction(t *testing.T) {
	game := &RemoteGame{action: matchMenuActionRoomMenu}
	if got := game.ConsumeAction(); got != matchMenuActionRoomMenu {
		t.Fatalf("expected room menu action, got %v", got)
	}
	if got := game.ConsumeAction(); got != matchMenuActionNone {
		t.Fatalf("expected action reset, got %v", got)
	}
}

func TestRemoteSyncMenuState(t *testing.T) {
	game := &RemoteGame{}
	game.disconnected = "Disconnected from server"
	game.syncMenuState()
	if game.menu.Mode != matchMenuModeDisconnected {
		t.Fatalf("expected disconnected menu, got %v", game.menu.Mode)
	}

	game.disconnected = ""
	game.state.GameOver = true
	game.syncMenuState()
	if game.menu.Mode != matchMenuModePostgame {
		t.Fatalf("expected postgame menu, got %v", game.menu.Mode)
	}

	game.state.GameOver = false
	game.syncMenuState()
	if game.menu.Mode != matchMenuModeHidden {
		t.Fatalf("expected menu to close, got %v", game.menu.Mode)
	}
}

func TestRemoteCurrentInputForPendingRematchVote(t *testing.T) {
	game := &RemoteGame{localTeam: sim.TeamAway, state: sim.GameState{Tick: 9, GameOver: true}, pendingRematchVote: true}
	input := game.currentInput()
	if input.Team != sim.TeamAway || input.Tick != 10 || !input.Ready {
		t.Fatalf("unexpected rematch input %+v", input)
	}
	if game.pendingRematchVote {
		t.Fatalf("expected pending rematch vote to clear after sending input")
	}
}

func TestRemoteCurrentInputStopsGameplayWhileMenuVisible(t *testing.T) {
	game := &RemoteGame{localTeam: sim.TeamHome, state: sim.GameState{Tick: 4, Phase: sim.MatchPhasePlaying}, menu: matchMenuState{Mode: matchMenuModePause}}
	input := game.currentInput()
	if input.Tick != 5 || input.Team != sim.TeamHome {
		t.Fatalf("unexpected input header %+v", input)
	}
	if input.Move != (sim.Vec2{}) || input.Shoot || input.Pass || input.Switch {
		t.Fatalf("expected no gameplay input while menu visible, got %+v", input)
	}
}

func TestRemoteMenuEntries(t *testing.T) {
	game := &RemoteGame{localTeam: sim.TeamHome}
	game.menu.Mode = matchMenuModePause
	entries := game.remoteMenuEntries()
	if len(entries) != 3 || entries[0] != (ui.MenuEntry{Label: "Resume"}) || entries[2] != (ui.MenuEntry{Label: "Room Menu"}) {
		t.Fatalf("unexpected pause entries %+v", entries)
	}

	game.standalone = true
	entries = game.remoteMenuEntries()
	if entries[2].Label != "Quit Match" {
		t.Fatalf("expected standalone room entry to become quit match, got %+v", entries[2])
	}

	game.menu.Mode = matchMenuModePostgame
	game.standalone = false
	game.pendingRematchVote = true
	entries = game.remoteMenuEntries()
	if !entries[0].Disabled || entries[0].Label != "Waiting for Other Player" {
		t.Fatalf("expected waiting play again entry, got %+v", entries[0])
	}

	game.menu.Mode = matchMenuModeDisconnected
	entries = game.remoteMenuEntries()
	if len(entries) != 2 || entries[0].Label != "Quit Match" {
		t.Fatalf("unexpected disconnected entries %+v", entries)
	}
}

func TestRemoteMenuTextAndStatus(t *testing.T) {
	game := &RemoteGame{localTeam: sim.TeamHome, roomCode: "AB12C", state: sim.GameState{Score: sim.Score{Home: 3, Away: 1}}}
	game.menu.Mode = matchMenuModePause
	if title, subtitle, footer := game.remoteMenuText(); title != "Match Menu" || subtitle == "" || footer == "" {
		t.Fatalf("unexpected pause menu text: %q %q %q", title, subtitle, footer)
	}
	if status := game.networkStatus(); status != "Match menu open  Choose Resume, Quit Match, or Room Menu" {
		t.Fatalf("unexpected pause status %q", status)
	}

	game.menu.Mode = matchMenuModePostgame
	if title, subtitle, footer := game.remoteMenuText(); title != "You Win" || subtitle == "" || footer == "" {
		t.Fatalf("unexpected win postgame text: %q %q %q", title, subtitle, footer)
	}
	game.state.HomeReady = true
	_, subtitle, _ := game.remoteMenuText()
	if subtitle != "Rematch requested. Waiting for the other player." {
		t.Fatalf("unexpected waiting subtitle %q", subtitle)
	}
	if status := game.networkStatus(); status != "Game over  Choose Play Again, Quit Match, or Room Menu" {
		t.Fatalf("unexpected postgame status %q", status)
	}

	game.menu.Mode = matchMenuModeDisconnected
	if title, subtitle, footer := game.remoteMenuText(); title != "Connection Lost" || subtitle == "" || footer == "" {
		t.Fatalf("unexpected disconnected text: %q %q %q", title, subtitle, footer)
	}
	if status := game.networkStatus(); status != "Disconnected from server  Choose Quit Match or Room Menu" {
		t.Fatalf("unexpected disconnected status %q", status)
	}
}

func TestRemoteHelpers(t *testing.T) {
	game := &RemoteGame{localTeam: sim.TeamHome, state: sim.GameState{HomeReady: true, AwayReady: false, Score: sim.Score{Home: 2, Away: 4}}}
	if got := game.roomMenuAction(); got != matchMenuActionRoomMenu {
		t.Fatalf("expected room menu action, got %v", got)
	}
	game.standalone = true
	if got := game.roomMenuAction(); got != matchMenuActionQuit {
		t.Fatalf("expected quit action in standalone mode, got %v", got)
	}
	if !game.localTeamReady() {
		t.Fatalf("expected home team ready")
	}
	if got := game.scoreFor(sim.TeamAway); got != 4 {
		t.Fatalf("expected away score 4, got %d", got)
	}
	if got := game.opponentTeam(); got != sim.TeamAway {
		t.Fatalf("expected away opponent, got %q", got)
	}
}

func TestNewSoloGameWithColorsSetsTeamColors(t *testing.T) {
	game := NewSoloGameWithColors(sim.TeamColorGreen, sim.TeamColorOrange)
	if game.state.HomeColor != sim.TeamColorGreen || game.state.AwayColor != sim.TeamColorOrange {
		t.Fatalf("unexpected solo colors: home=%q away=%q", game.state.HomeColor, game.state.AwayColor)
	}
}

func TestSoloConsumeActionClearsAction(t *testing.T) {
	game := NewSoloGame()
	game.action = matchMenuActionQuit
	if got := game.ConsumeAction(); got != matchMenuActionQuit {
		t.Fatalf("expected quit action, got %v", got)
	}
	if got := game.ConsumeAction(); got != matchMenuActionNone {
		t.Fatalf("expected action to clear, got %v", got)
	}
}

func TestSoloSyncMenuStateTracksPostgame(t *testing.T) {
	game := NewSoloGame()
	game.state.GameOver = true
	game.syncMenuState()
	if game.menu.Mode != matchMenuModePostgame {
		t.Fatalf("expected postgame menu, got %v", game.menu.Mode)
	}

	game.state.GameOver = false
	game.syncMenuState()
	if game.menu.Mode != matchMenuModeHidden {
		t.Fatalf("expected menu to close after leaving postgame, got %v", game.menu.Mode)
	}
}

func TestSoloRestartMatchPreservesColorsAndResetsMenu(t *testing.T) {
	game := NewSoloGameWithColors(sim.TeamColorBlack, sim.TeamColorRed)
	game.state.Score.Home = 3
	game.menu.Open(matchMenuModePause)
	game.restartMatch()

	if game.state.Score.Home != 0 || game.state.Period != 1 {
		t.Fatalf("expected restarted solo match, got score=%+v period=%d", game.state.Score, game.state.Period)
	}
	if game.state.HomeColor != sim.TeamColorBlack || game.state.AwayColor != sim.TeamColorRed {
		t.Fatalf("expected colors preserved, got home=%q away=%q", game.state.HomeColor, game.state.AwayColor)
	}
	if game.menu.Visible() {
		t.Fatalf("expected menu to close on restart")
	}
}

func TestSoloMatchMenuContent(t *testing.T) {
	game := NewSoloGame()
	game.menu.Mode = matchMenuModePause
	pauseTitle, pauseSubtitle, pauseFooter, pauseEntries := game.matchMenuContent()
	if pauseTitle != "Pause Menu" || pauseSubtitle == "" || pauseFooter == "" || len(pauseEntries) != 3 {
		t.Fatalf("unexpected pause menu content: %q %q %q %+v", pauseTitle, pauseSubtitle, pauseFooter, pauseEntries)
	}

	game.menu.Mode = matchMenuModePostgame
	postTitle, postSubtitle, postFooter, postEntries := game.matchMenuContent()
	if postTitle == "" || postSubtitle != "The match is over." || postFooter == "" || len(postEntries) != 2 {
		t.Fatalf("unexpected postgame menu content: %q %q %q %+v", postTitle, postSubtitle, postFooter, postEntries)
	}

	game.menu.Mode = matchMenuModeHidden
	emptyTitle, emptySubtitle, emptyFooter, emptyEntries := game.matchMenuContent()
	if emptyTitle != "" || emptySubtitle != "" || emptyFooter != "" || emptyEntries != nil {
		t.Fatalf("expected empty hidden menu content, got %q %q %q %+v", emptyTitle, emptySubtitle, emptyFooter, emptyEntries)
	}
}

func TestSoloStatusAndLabels(t *testing.T) {
	game := NewSoloGame()
	if status := game.soloStatus(); status == "" {
		t.Fatalf("expected non-empty default solo status")
	}
	game.menu.Mode = matchMenuModePause
	if status := game.soloStatus(); status != "Paused  Choose Resume, Restart Match, or Quit" {
		t.Fatalf("unexpected pause status %q", status)
	}
	game.menu.Mode = matchMenuModePostgame
	if status := game.soloStatus(); status != "Game over  Choose Play Again or Quit" {
		t.Fatalf("unexpected postgame status %q", status)
	}
	if game.resumeLabel() != "Resume" || game.restartLabel() != "Restart Match" || game.playAgainLabel() != "Play Again" {
		t.Fatalf("unexpected static menu labels")
	}
	if game.quitLabel() != "Quit to Launcher" {
		t.Fatalf("expected launcher quit label, got %q", game.quitLabel())
	}
	game.standalone = true
	if game.quitLabel() != "Quit Game" {
		t.Fatalf("expected standalone quit label, got %q", game.quitLabel())
	}
}

func TestSoloPostgameTitle(t *testing.T) {
	game := NewSoloGame()
	game.state.Score = sim.Score{Home: 3, Away: 1}
	if title := game.postgameTitle(); title != "You Win" {
		t.Fatalf("expected win title, got %q", title)
	}
	game.state.Score = sim.Score{Home: 1, Away: 3}
	if title := game.postgameTitle(); title != "You Lose" {
		t.Fatalf("expected lose title, got %q", title)
	}
	game.state.Score = sim.Score{Home: 2, Away: 2}
	if title := game.postgameTitle(); title != "Game Over" {
		t.Fatalf("expected tie title, got %q", title)
	}
}

func TestSoloMatchMenuEntriesMirrorsContent(t *testing.T) {
	game := NewSoloGame()
	game.menu.Mode = matchMenuModePause
	entries := game.matchMenuEntries()
	if len(entries) != 3 || entries[0] != (ui.MenuEntry{Label: "Resume"}) {
		t.Fatalf("unexpected menu entries %+v", entries)
	}
}

func TestNewSoloGameEnablesMenus(t *testing.T) {
	game := NewSoloGame()
	if !game.state.UseMenus {
		t.Fatalf("expected solo game to enable menu phases")
	}
}

func TestSoloSyncMenuStateTracksIntermission(t *testing.T) {
	game := NewSoloGame()
	game.state.Phase = sim.MatchPhaseIntermission
	game.state.LastIntermissionStats = sim.PeriodStats{Period: 1, Home: sim.TeamPeriodStats{ShotsOnGoal: 6, Goals: 2}, Away: sim.TeamPeriodStats{ShotsOnGoal: 4, Goals: 1}}
	game.syncMenuState()
	if game.menu.Mode != matchMenuModeIntermission {
		t.Fatalf("expected intermission menu, got %v", game.menu.Mode)
	}

	game.continueIntermission()
	if game.menu.Mode != matchMenuModeHidden {
		t.Fatalf("expected menu to close after continuing, got %v", game.menu.Mode)
	}
	if game.state.Phase != sim.MatchPhasePlaying {
		t.Fatalf("expected match to resume playing, got %q", game.state.Phase)
	}
}

func TestSoloIntermissionMenuContentAndStatus(t *testing.T) {
	game := NewSoloGame()
	game.state.Phase = sim.MatchPhaseIntermission
	game.state.LastIntermissionStats = sim.PeriodStats{Period: 2, Home: sim.TeamPeriodStats{ShotsOnGoal: 8, Goals: 3}, Away: sim.TeamPeriodStats{ShotsOnGoal: 5, Goals: 2}}
	game.menu.Mode = matchMenuModeIntermission

	title, subtitle, footer, entries := game.matchMenuContent()
	if title != "End of Period 2" || subtitle == "" || footer == "" || len(entries) != 3 {
		t.Fatalf("unexpected intermission menu content: %q %q %q %+v", title, subtitle, footer, entries)
	}
	if entries[0].Label != "Continue" {
		t.Fatalf("expected continue entry first, got %+v", entries)
	}
	if status := game.soloStatus(); status != "Intermission  Choose Continue, Restart Match, or Quit" {
		t.Fatalf("unexpected intermission status %q", status)
	}
}

func TestSetOnlineRoomsPreservesSelectedRoom(t *testing.T) {
	app := &App{menu: launchMenu{OnlineRooms: []netcode.RoomSummary{{Code: "AB12C", Name: "First"}, {Code: "CD34E", Name: "Second"}}, OnlineRoomCursor: 1, OnlineFocus: onlineFieldRoomList}}
	app.setOnlineRooms([]netcode.RoomSummary{{Code: "ZZ99Z", Name: "Other"}, {Code: "CD34E", Name: "Second"}})
	if app.menu.OnlineRoomCursor != 1 {
		t.Fatalf("expected online cursor to follow matching room, got %d", app.menu.OnlineRoomCursor)
	}

	app.setOnlineRooms(nil)
	if app.menu.OnlineRoomCursor != 0 || app.menu.OnlineFocus != onlineFieldRoomName {
		t.Fatalf("expected empty online list to reset cursor/focus, got cursor=%d focus=%v", app.menu.OnlineRoomCursor, app.menu.OnlineFocus)
	}
}

func TestJoinOnlineListedRoomStatusGuards(t *testing.T) {
	app := &App{}
	if err := app.joinOnlineListedRoom(0); err != nil {
		t.Fatalf("join listed room without rooms: %v", err)
	}
	if app.menu.Status != "No online rooms are open right now" {
		t.Fatalf("unexpected empty online room status %q", app.menu.Status)
	}

	app.menu.OnlineRooms = []netcode.RoomSummary{{Code: "ABCDE", Name: "Full Room", Players: 2, Capacity: 2}}
	if err := app.joinOnlineListedRoom(0); err != nil {
		t.Fatalf("join full listed room: %v", err)
	}
	if app.menu.Status != "That room is already full" {
		t.Fatalf("unexpected full online room status %q", app.menu.Status)
	}
	if app.menu.OnlineRoomCode != "ABCDE" {
		t.Fatalf("expected selected online room code to populate, got %q", app.menu.OnlineRoomCode)
	}
}
