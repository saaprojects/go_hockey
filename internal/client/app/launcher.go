package app

import (
	"fmt"

	"hockeyv2/internal/client/render"
	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/netcode"
	"hockeyv2/internal/server"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type App struct {
	screen         appScreen
	menu           launchMenu
	setup          launchSetupState
	solo           *SoloGame
	remote         *RemoteGame
	hostServer     *server.Server
	hostServeErr   chan error
	hostAdvertiser *discovery.Advertiser
	browser        *discovery.Browser
}

func NewApp() *App {
	app := &App{
		screen: appScreenMenu,
		menu: launchMenu{
			Selected: menuOptionSolo,
			Color:    sim.TeamColorBlue,
		},
	}
	browser, err := discovery.NewBrowser()
	if err != nil {
		app.menu.Status = "LAN discovery unavailable"
		return app
	}
	app.browser = browser
	return app
}

func (a *App) Close() error {
	a.stopMenuMusic()
	a.stopMatchAmbience()
	if a.remote != nil {
		_ = a.remote.Close()
	}
	a.stopHostedServer()
	if a.browser != nil {
		_ = a.browser.Close()
		a.browser = nil
	}
	return nil
}

func (a *App) stopMatchAmbience() {
	if a.solo != nil && a.solo.sounds != nil {
		a.solo.sounds.StopArenaAmbience()
	}
	if a.remote != nil && a.remote.sounds != nil {
		a.remote.sounds.StopArenaAmbience()
	}
}

func (a *App) stopMenuMusic() {
	defaultSoundboard().StopMenuMusic()
}

func (a *App) syncScreenAudio() {
	sounds := defaultSoundboard()
	switch a.screen {
	case appScreenSolo, appScreenRemote:
		sounds.StopMenuMusic()
	default:
		sounds.PlayMenuMusic()
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(sim.WindowWidth), int(sim.WindowHeight)
}

func (a *App) Update() error {
	a.pollDiscoveryUpdates()
	a.syncScreenAudio()
	switch a.screen {
	case appScreenSolo:
		if err := a.solo.Update(); err != nil {
			return err
		}
		if a.solo.ConsumeAction() == matchMenuActionQuit {
			a.returnToMenu("")
		}
		return nil
	case appScreenRemote:
		select {
		case err := <-a.hostServeErr:
			if err != nil {
				a.returnToMenu("Host stopped")
				return nil
			}
		default:
		}
		if err := a.remote.Update(); err != nil {
			return err
		}
		status := a.remote.disconnected
		switch a.remote.ConsumeAction() {
		case matchMenuActionQuit:
			a.returnToMenu(status)
		case matchMenuActionRoomMenu:
			a.returnToRoomMenu(status)
		}
		return nil
	case appScreenJoinBrowser:
		return a.updateJoinBrowser()
	default:
		return a.updateMenu()
	}
}

func (a *App) Draw(screen *ebiten.Image) {
	switch a.screen {
	case appScreenSolo:
		a.solo.Draw(screen)
	case appScreenRemote:
		a.remote.Draw(screen)
	case appScreenJoinBrowser:
		render.DrawJoinBrowser(screen, render.JoinBrowserModel{Rooms: a.menu.Rooms, SelectedRoom: a.menu.RoomCursor, Status: a.menu.Status})
	default:
		render.DrawLauncherMenu(screen, render.LauncherMenuModel{SelectedOption: int(a.menu.Selected), Status: a.menu.Status, RoomCount: len(a.menu.Rooms)})
		if a.setup.Active {
			render.DrawLaunchSetup(screen, a.launchSetupModel())
		}
	}
}

func (a *App) launchSetupModel() render.LaunchSetupModel {
	model := render.LaunchSetupModel{
		ModeLabel:    "Solo Game Setup",
		Description:  "Pick your team color, then start a local AI match.",
		ConfirmLabel: "Start Solo Game",
		Color:        a.setup.Color,
		Status:       a.menu.Status,
	}
	if a.setup.Mode == menuOptionHost {
		model.ModeLabel = "Host Multiplayer Setup"
		model.Description = "Pick your team color, then start a LAN room from this client."
		model.ConfirmLabel = "Host LAN Game"
	}
	return model
}

func (a *App) pollDiscoveryUpdates() {
	if a.browser == nil {
		return
	}
	for {
		select {
		case rooms, ok := <-a.browser.Updates():
			if !ok {
				a.browser = nil
				return
			}
			a.setDiscoveredRooms(rooms)
		default:
			return
		}
	}
}

func (a *App) setDiscoveredRooms(rooms []discovery.Room) {
	selectedKey := ""
	if len(a.menu.Rooms) > 0 && a.menu.RoomCursor >= 0 && a.menu.RoomCursor < len(a.menu.Rooms) {
		selectedKey = roomKey(a.menu.Rooms[a.menu.RoomCursor])
	}
	a.menu.Rooms = rooms
	if len(a.menu.Rooms) == 0 {
		a.menu.RoomCursor = 0
		return
	}
	if selectedKey != "" {
		for index, room := range a.menu.Rooms {
			if roomKey(room) == selectedKey {
				a.menu.RoomCursor = index
				return
			}
		}
	}
	if a.menu.RoomCursor < 0 {
		a.menu.RoomCursor = 0
	}
	if a.menu.RoomCursor >= len(a.menu.Rooms) {
		a.menu.RoomCursor = len(a.menu.Rooms) - 1
	}
}

func (a *App) updateMenu() error {
	if a.setup.Active {
		return a.updateLaunchSetup()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		x := float64(cursorX)
		y := float64(cursorY)
		for index := 0; index < 3; index++ {
			if !ui.PointInRect(x, y, render.MenuOptionRect(index)) {
				continue
			}
			a.menu.Selected = menuOption(index)
			return a.activateMenuOption(a.menu.Selected)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		a.menu.Selected = (a.menu.Selected + 2) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		a.menu.Selected = (a.menu.Selected + 1) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return a.activateMenuOption(a.menu.Selected)
	}
	return nil
}

func (a *App) updateLaunchSetup() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.closeLaunchSetup()
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		x := float64(cursorX)
		y := float64(cursorY)
		for index, teamColor := range launcherColorCycle {
			if ui.PointInRect(x, y, render.LaunchSetupColorChipRect(index)) {
				a.setup.Color = teamColor
				return nil
			}
		}
		if ui.PointInRect(x, y, render.LaunchSetupBackRect()) {
			a.closeLaunchSetup()
			return nil
		}
		if ui.PointInRect(x, y, render.LaunchSetupConfirmRect()) {
			return a.confirmLaunchSetup()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		a.setup.Color = nextLauncherColor(a.setup.Color, -1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		a.setup.Color = nextLauncherColor(a.setup.Color, 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return a.confirmLaunchSetup()
	}
	return nil
}

func (a *App) activateMenuOption(option menuOption) error {
	a.menu.Selected = option
	switch option {
	case menuOptionSolo, menuOptionHost:
		a.openLaunchSetup(option)
	case menuOptionJoin:
		a.closeLaunchSetup()
		a.screen = appScreenJoinBrowser
		if a.browser == nil {
			a.menu.Status = "LAN discovery unavailable"
		} else if len(a.menu.Rooms) == 0 {
			a.menu.Status = "Searching for LAN rooms"
		} else {
			a.menu.Status = ""
		}
		ebiten.SetWindowTitle("Go Hockey - Join LAN Room")
	}
	return nil
}

func (a *App) openLaunchSetup(option menuOption) {
	a.setup.Open(option, a.menu.Color)
	a.menu.Status = ""
}

func (a *App) closeLaunchSetup() {
	a.setup.Close()
}

func (a *App) confirmLaunchSetup() error {
	a.menu.Color = a.setup.Color
	switch a.setup.Mode {
	case menuOptionSolo:
		a.solo = NewSoloGameWithColors(a.menu.Color, opponentColorForSelection(a.menu.Color))
		a.remote = nil
		a.screen = appScreenSolo
		a.menu.Status = ""
		a.closeLaunchSetup()
		ebiten.SetWindowTitle("Go Hockey - Solo")
	case menuOptionHost:
		if err := a.startHostedRemote(":4242", a.menu.Color); err != nil {
			a.menu.Status = "Unable to advertise local room"
			return nil
		}
		a.closeLaunchSetup()
	}
	return nil
}

func (a *App) updateJoinBrowser() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.screen = appScreenMenu
		a.menu.Status = ""
		ebiten.SetWindowTitle("Go Hockey")
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if roomIndex, ok := a.joinRoomAtCursor(); ok {
			a.menu.RoomCursor = roomIndex
			return a.joinRoom(roomIndex)
		}
	}
	if len(a.menu.Rooms) > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
			a.menu.RoomCursor = (a.menu.RoomCursor + len(a.menu.Rooms) - 1) % len(a.menu.Rooms)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
			a.menu.RoomCursor = (a.menu.RoomCursor + 1) % len(a.menu.Rooms)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return a.joinRoom(a.menu.RoomCursor)
	}
	return nil
}

func (a *App) joinRoom(index int) error {
	if a.browser == nil {
		a.menu.Status = "LAN discovery unavailable"
		return nil
	}
	if len(a.menu.Rooms) == 0 {
		a.menu.Status = "Searching for LAN rooms"
		return nil
	}
	if index < 0 || index >= len(a.menu.Rooms) {
		return nil
	}
	room := a.menu.Rooms[index]
	if !room.Joinable() {
		a.menu.Status = "That room is already full"
		return nil
	}
	if err := a.startRemote(room.Addr); err != nil {
		a.menu.Status = "Unable to connect to room"
	}
	return nil
}

func (a *App) startHostedRemote(listenAddr string, homeColor sim.TeamColor) error {
	a.stopHostedServer()
	srv, serveErr, advertiser, game, err := startHostedSession(listenAddr, homeColor)
	if err != nil {
		return err
	}
	a.hostServer = srv
	a.hostServeErr = serveErr
	a.hostAdvertiser = advertiser
	a.remote = game
	a.solo = nil
	a.screen = appScreenRemote
	a.menu.Status = fmt.Sprintf("Hosting room %s", advertiser.Room().Code)
	ebiten.SetWindowTitle("Go Hockey - Host Multiplayer")
	return nil
}

func (a *App) startRemote(addr string) error {
	a.stopHostedServer()
	clientConn, err := netcode.Dial(addr)
	if err != nil {
		return err
	}
	a.remote = newRemoteGame(clientConn)
	a.solo = nil
	a.screen = appScreenRemote
	a.hostServeErr = nil
	a.menu.Status = "Connected to room"
	ebiten.SetWindowTitle(remoteWindowTitle(string(a.remote.localTeam)))
	return nil
}

func (a *App) returnToMenu(status string) {
	a.stopMatchAmbience()
	if a.remote != nil {
		_ = a.remote.Close()
	}
	a.remote = nil
	a.solo = nil
	a.stopHostedServer()
	a.closeLaunchSetup()
	a.screen = appScreenMenu
	a.menu.Status = status
	ebiten.SetWindowTitle("Go Hockey")
}

func (a *App) returnToRoomMenu(status string) {
	a.stopMatchAmbience()
	if a.remote != nil {
		_ = a.remote.Close()
	}
	a.remote = nil
	a.solo = nil
	a.stopHostedServer()
	a.closeLaunchSetup()
	if a.browser == nil {
		a.screen = appScreenMenu
		if status == "" {
			status = "LAN discovery unavailable"
		}
		a.menu.Status = status
		ebiten.SetWindowTitle("Go Hockey")
		return
	}
	a.screen = appScreenJoinBrowser
	if status != "" {
		a.menu.Status = status
	} else if len(a.menu.Rooms) == 0 {
		a.menu.Status = "Searching for LAN rooms"
	} else {
		a.menu.Status = ""
	}
	ebiten.SetWindowTitle("Go Hockey - Join LAN Room")
}

func (a *App) stopHostedServer() {
	if a.hostAdvertiser != nil {
		_ = a.hostAdvertiser.Close()
		a.hostAdvertiser = nil
	}
	if a.hostServer != nil {
		_ = a.hostServer.Close()
		a.hostServer = nil
	}
	a.hostServeErr = nil
}

func (a *App) joinRoomAtCursor() (int, bool) {
	cursorX, cursorY := ebiten.CursorPosition()
	for _, card := range render.JoinRoomCards(len(a.menu.Rooms), a.menu.RoomCursor) {
		if ui.PointInRect(float64(cursorX), float64(cursorY), card.Area) {
			return card.Index, true
		}
	}
	return 0, false
}
