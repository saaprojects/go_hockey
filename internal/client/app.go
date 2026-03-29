package client

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"hockeyv2/internal/discovery"
	"hockeyv2/internal/netcode"
	"hockeyv2/internal/server"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type appScreen int

type menuOption int

type rect struct {
	x float64
	y float64
	w float64
	h float64
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

var launcherColorCycle = []sim.TeamColor{
	sim.TeamColorBlack,
	sim.TeamColorOrange,
	sim.TeamColorGreen,
	sim.TeamColorBlue,
	sim.TeamColorRed,
}

type App struct {
	screen         appScreen
	menu           launchMenu
	solo           *SoloGame
	remote         *RemoteGame
	hostServer     *server.Server
	hostServeErr   chan error
	hostAdvertiser *discovery.Advertiser
	browser        *discovery.Browser
}

type launchMenu struct {
	selected   menuOption
	soloColor  sim.TeamColor
	status     string
	rooms      []discovery.Room
	roomCursor int
}

func RunApp() error {
	app := NewApp()
	defer app.Close()
	defer ebiten.SetWindowTitle("Go Hockey")
	defer ebiten.SetTPS(sim.TickRate)
	defer ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))

	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle("Go Hockey")
	ebiten.SetTPS(sim.TickRate)
	return ebiten.RunGame(app)
}

func NewApp() *App {
	app := &App{
		screen: appScreenMenu,
		menu: launchMenu{
			selected:  menuOptionSolo,
			soloColor: sim.TeamColorBlue,
		},
	}
	browser, err := discovery.NewBrowser()
	if err != nil {
		app.menu.status = "LAN discovery unavailable"
		return app
	}
	app.browser = browser
	return app
}

func (a *App) Close() error {
	if a.remote != nil && a.remote.client != nil {
		_ = a.remote.client.Close()
	}
	a.stopHostedServer()
	if a.browser != nil {
		_ = a.browser.Close()
		a.browser = nil
	}
	return nil
}

func (a *App) Update() error {
	a.pollDiscoveryUpdates()
	switch a.screen {
	case appScreenSolo:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.returnToMenu("")
			return nil
		}
		return a.solo.Update()
	case appScreenRemote:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.returnToMenu("")
			return nil
		}
		select {
		case err := <-a.hostServeErr:
			if err != nil {
				a.returnToMenu("Host stopped")
				return nil
			}
		default:
		}
		return a.remote.Update()
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
		a.drawJoinBrowser(screen)
	default:
		a.drawMenu(screen)
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(sim.WindowWidth), int(sim.WindowHeight)
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
	if len(a.menu.rooms) > 0 && a.menu.roomCursor >= 0 && a.menu.roomCursor < len(a.menu.rooms) {
		selectedKey = roomKey(a.menu.rooms[a.menu.roomCursor])
	}
	a.menu.rooms = rooms
	if len(a.menu.rooms) == 0 {
		a.menu.roomCursor = 0
		return
	}
	if selectedKey != "" {
		for index, room := range a.menu.rooms {
			if roomKey(room) == selectedKey {
				a.menu.roomCursor = index
				return
			}
		}
	}
	if a.menu.roomCursor < 0 {
		a.menu.roomCursor = 0
	}
	if a.menu.roomCursor >= len(a.menu.rooms) {
		a.menu.roomCursor = len(a.menu.rooms) - 1
	}
}

func (a *App) updateMenu() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		x := float64(cursorX)
		y := float64(cursorY)
		if a.menu.selected == menuOptionSolo {
			if pointInRect(x, y, launcherSoloColorPrevRect()) {
				a.menu.soloColor = nextLauncherColor(a.menu.soloColor, -1)
				return nil
			}
			if pointInRect(x, y, launcherSoloColorNextRect()) {
				a.menu.soloColor = nextLauncherColor(a.menu.soloColor, 1)
				return nil
			}
		}
		if option, ok := a.menuOptionAtCursor(); ok {
			a.menu.selected = option
			return a.activateMenuOption(option)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		a.menu.selected = (a.menu.selected + 2) % 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		a.menu.selected = (a.menu.selected + 1) % 3
	}
	if a.menu.selected == menuOptionSolo {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			a.menu.soloColor = nextLauncherColor(a.menu.soloColor, -1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			a.menu.soloColor = nextLauncherColor(a.menu.soloColor, 1)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return a.activateMenuOption(a.menu.selected)
	}
	return nil
}

func (a *App) activateMenuOption(option menuOption) error {
	switch option {
	case menuOptionSolo:
		a.solo = NewSoloGameWithColors(a.menu.soloColor, awayColorForSolo(a.menu.soloColor))
		a.screen = appScreenSolo
		a.menu.status = ""
		ebiten.SetWindowTitle("Go Hockey - Solo")
	case menuOptionHost:
		if err := a.startHostedRemote(":4242"); err != nil {
			a.menu.status = "Unable to advertise local room"
		}
	case menuOptionJoin:
		a.screen = appScreenJoinBrowser
		if a.browser == nil {
			a.menu.status = "LAN discovery unavailable"
		} else if len(a.menu.rooms) == 0 {
			a.menu.status = "Searching for LAN rooms"
		} else {
			a.menu.status = ""
		}
		ebiten.SetWindowTitle("Go Hockey - Join LAN Room")
	}
	return nil
}

func (a *App) updateJoinBrowser() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.screen = appScreenMenu
		a.menu.status = ""
		ebiten.SetWindowTitle("Go Hockey")
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if roomIndex, ok := a.joinRoomAtCursor(); ok {
			a.menu.roomCursor = roomIndex
			return a.joinRoom(roomIndex)
		}
	}
	if len(a.menu.rooms) > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
			a.menu.roomCursor = (a.menu.roomCursor + len(a.menu.rooms) - 1) % len(a.menu.rooms)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
			a.menu.roomCursor = (a.menu.roomCursor + 1) % len(a.menu.rooms)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return a.joinRoom(a.menu.roomCursor)
	}
	return nil
}

func (a *App) joinRoom(index int) error {
	if a.browser == nil {
		a.menu.status = "LAN discovery unavailable"
		return nil
	}
	if len(a.menu.rooms) == 0 {
		a.menu.status = "Searching for LAN rooms"
		return nil
	}
	if index < 0 || index >= len(a.menu.rooms) {
		return nil
	}
	room := a.menu.rooms[index]
	if !room.Joinable() {
		a.menu.status = "That room is already full"
		return nil
	}
	if err := a.startRemote(room.Addr); err != nil {
		a.menu.status = "Unable to connect to room"
	}
	return nil
}

func (a *App) drawMenu(screen *ebiten.Image) {
	screen.Fill(colorHUDBackground)
	drawRoundedFill(screen, sim.RinkLeft-16, sim.RinkTop-16, sim.RinkRight-sim.RinkLeft+32, sim.RinkBottom-sim.RinkTop+32, sim.RinkCornerRadius+16, colorBoard)
	drawRoundedFill(screen, sim.RinkLeft-10, sim.RinkTop-10, sim.RinkRight-sim.RinkLeft+20, sim.RinkBottom-sim.RinkTop+20, sim.RinkCornerRadius+10, colorBoardOutline)
	drawRoundedFill(screen, sim.RinkLeft, sim.RinkTop, sim.RinkRight-sim.RinkLeft, sim.RinkBottom-sim.RinkTop, sim.RinkCornerRadius, colorIce)

	topPanelX := sim.CenterX - 290.0
	topPanelY := 42.0
	topPanelWidth := 580.0
	topPanelHeight := 102.0
	footerX := sim.CenterX - 290.0
	footerY := 536.0
	footerWidth := 580.0
	footerHeight := 112.0

	topFill := color.RGBA{0xf8, 0xfb, 0xff, 0xff}
	footerFill := color.RGBA{0x13, 0x29, 0x44, 0xff}
	cardFill := color.RGBA{0xfc, 0xfd, 0xff, 0xff}
	selectedFill := color.RGBA{0x1c, 0x42, 0x6b, 0xff}
	mutedText := color.RGBA{0x5b, 0x6c, 0x80, 0xff}
	lightText := color.RGBA{0xee, 0xf5, 0xff, 0xff}
	accentText := color.RGBA{0xa9, 0xd3, 0xff, 0xff}
	inputFill := color.RGBA{0x09, 0x18, 0x2a, 0xff}

	ebitenutil.DrawRect(screen, topPanelX+8, topPanelY+10, topPanelWidth, topPanelHeight, colorPanelShadow)
	drawRoundedFill(screen, topPanelX, topPanelY, topPanelWidth, topPanelHeight, 24, topFill)
	title := "Go Hockey"
	titleWidth, _ := measureUIText(title, uiTitleFace)
	drawUIText(screen, title, uiTitleFace, sim.CenterX-titleWidth/2, 60, colorTextDark)
	subtitle := "Choose solo, host, or browse LAN rooms from this same client"
	subtitleWidth, _ := measureUIText(subtitle, uiBodyFace)
	drawUIText(screen, subtitle, uiBodyFace, sim.CenterX-subtitleWidth/2, 96, colorTextDark)
	controls := "Click or press Enter to start. Up or Down changes mode. Esc returns here from a match"
	controlsWidth, _ := measureUIText(controls, uiSmallFace)
	drawUIText(screen, controls, uiSmallFace, sim.CenterX-controlsWidth/2, 122, mutedText)

	labels := []string{"Solo Game", "Host Multiplayer", "Join Multiplayer"}
	details := []string{
		"Play locally against AI in the same client.",
		"Start a local server and advertise it on your LAN.",
		"Browse LAN rooms nearby and click one to connect.",
	}
	for index, label := range labels {
		rect := menuOptionRect(index)
		a.drawMenuOptionCard(screen, rect.x, rect.y, rect.w, rect.h, label, details[index], a.menu.selected == menuOption(index), cardFill, selectedFill)
	}

	ebitenutil.DrawRect(screen, footerX+8, footerY+10, footerWidth, footerHeight, color.RGBA{0x03, 0x0b, 0x14, 0x44})
	drawRoundedFill(screen, footerX, footerY, footerWidth, footerHeight, 20, footerFill)
	colorsLine := "Team colors: Black  |  Orange  |  Green  |  Blue  |  Red"
	colorsWidth, _ := measureUIText(colorsLine, uiBodyFace)
	drawUIText(screen, colorsLine, uiBodyFace, sim.CenterX-colorsWidth/2, footerY+18, lightText)
	a.drawMenuFooter(screen, footerX, footerY, footerWidth, footerHeight, accentText, lightText, inputFill)
	if a.menu.status != "" {
		statusWidth, _ := measureUIText(a.menu.status, uiSmallFace)
		drawUIText(screen, a.menu.status, uiSmallFace, sim.CenterX-statusWidth/2, footerY+footerHeight+18, colorTextDark)
	}
}

func (a *App) drawJoinBrowser(screen *ebiten.Image) {
	screen.Fill(colorHUDBackground)
	drawRoundedFill(screen, sim.RinkLeft-16, sim.RinkTop-16, sim.RinkRight-sim.RinkLeft+32, sim.RinkBottom-sim.RinkTop+32, sim.RinkCornerRadius+16, colorBoard)
	drawRoundedFill(screen, sim.RinkLeft-10, sim.RinkTop-10, sim.RinkRight-sim.RinkLeft+20, sim.RinkBottom-sim.RinkTop+20, sim.RinkCornerRadius+10, colorBoardOutline)
	drawRoundedFill(screen, sim.RinkLeft, sim.RinkTop, sim.RinkRight-sim.RinkLeft, sim.RinkBottom-sim.RinkTop, sim.RinkCornerRadius, colorIce)

	panelX := sim.CenterX - 314.0
	panelY := 54.0
	panelWidth := 628.0
	panelHeight := 560.0
	headFill := color.RGBA{0xf7, 0xfb, 0xff, 0xff}
	bodyFill := color.RGBA{0x12, 0x28, 0x44, 0xff}
	bodyOutline := color.RGBA{0x38, 0x68, 0x99, 0xff}
	lightText := color.RGBA{0xee, 0xf5, 0xff, 0xff}
	accentText := color.RGBA{0xa9, 0xd3, 0xff, 0xff}
	mutedText := color.RGBA{0x6b, 0x7b, 0x8f, 0xff}

	ebitenutil.DrawRect(screen, panelX+10, panelY+12, panelWidth, panelHeight, colorPanelShadow)
	drawRoundedFill(screen, panelX, panelY, panelWidth, 120, 26, headFill)
	drawRoundedFill(screen, panelX, panelY+98, panelWidth, panelHeight-98, 26, bodyFill)
	ebitenutil.DrawRect(screen, panelX+26, panelY+140, panelWidth-52, 2, bodyOutline)

	title := "Join LAN Room"
	titleWidth, _ := measureUIText(title, uiTitleFace)
	drawUIText(screen, title, uiTitleFace, sim.CenterX-titleWidth/2, panelY+26, colorTextDark)
	subtitle := "Nearby hosts appear automatically. Click a room or press Enter to connect."
	subtitleWidth, _ := measureUIText(subtitle, uiBodyFace)
	drawUIText(screen, subtitle, uiBodyFace, sim.CenterX-subtitleWidth/2, panelY+62, colorTextDark)
	help := "Esc returns to the launcher. Up or Down changes rooms while you browse."
	helpWidth, _ := measureUIText(help, uiSmallFace)
	drawUIText(screen, help, uiSmallFace, sim.CenterX-helpWidth/2, panelY+92, mutedText)

	drawUIText(screen, "Available Rooms", uiBodyFace, panelX+34, panelY+120, accentText)
	if len(a.menu.rooms) == 0 {
		drawRoundedFill(screen, panelX+34, panelY+168, panelWidth-68, 122, 18, color.RGBA{0x19, 0x34, 0x55, 0xff})
		message := "Searching your LAN for open rooms..."
		messageWidth, _ := measureUIText(message, uiBodyFace)
		drawUIText(screen, message, uiBodyFace, sim.CenterX-messageWidth/2, panelY+212, lightText)
		secondary := "Have a friend click Host Multiplayer on this same network."
		secondaryWidth, _ := measureUIText(secondary, uiSmallFace)
		drawUIText(screen, secondary, uiSmallFace, sim.CenterX-secondaryWidth/2, panelY+248, accentText)
	} else {
		cursorX, cursorY := ebiten.CursorPosition()
		for _, card := range a.joinRoomCards() {
			room := a.menu.rooms[card.index]
			hovered := pointInRect(float64(cursorX), float64(cursorY), card.rect)
			selected := card.index == a.menu.roomCursor
			a.drawRoomCard(screen, card.rect, room, selected, hovered)
		}
	}

	if a.menu.status != "" {
		statusWidth, _ := measureUIText(a.menu.status, uiSmallFace)
		drawUIText(screen, a.menu.status, uiSmallFace, sim.CenterX-statusWidth/2, panelY+panelHeight-24, lightText)
	}
}

func (a *App) drawMenuOptionCard(screen *ebiten.Image, x, y, width, height float64, label, detail string, selected bool, cardFill, selectedFill color.RGBA) {
	fill := cardFill
	stripe := color.RGBA{0x4e, 0x72, 0x97, 0xff}
	titleColor := colorTextDark
	detailColor := color.RGBA{0x4f, 0x60, 0x74, 0xff}
	shadow := colorPanelShadow
	if selected {
		fill = selectedFill
		stripe = color.RGBA{0x46, 0x9b, 0xff, 0xff}
		titleColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
		detailColor = color.RGBA{0xdd, 0xee, 0xff, 0xff}
		shadow = color.RGBA{0x02, 0x08, 0x11, 0x60}
	}
	ebitenutil.DrawRect(screen, x+8, y+10, width, height, shadow)
	drawRoundedFill(screen, x, y, width, height, 20, fill)
	ebitenutil.DrawRect(screen, x, y, 18, height, stripe)
	drawUIText(screen, label, uiBodyFace, x+36, y+24, titleColor)
	drawUIText(screen, detail, uiSmallFace, x+36, y+58, detailColor)
}

func (a *App) drawMenuFooter(screen *ebiten.Image, x, y, width, height float64, accentText, lightText, inputFill color.RGBA) {
	titleY := y + 46
	rowY := y + 68
	helpY := y + 94
	switch a.menu.selected {
	case menuOptionSolo:
		palette := paletteForTeamColor(a.menu.soloColor)
		cursorX, cursorY := ebiten.CursorPosition()
		drawUIText(screen, "Solo team color", uiSmallFace, x+34, titleY, accentText)
		prevRect := launcherSoloColorPrevRect()
		nextRect := launcherSoloColorNextRect()
		labelRect := launcherSoloColorLabelRect()
		drawOverlayButton(screen, prevRect, "<", pointInRect(float64(cursorX), float64(cursorY), prevRect), false)
		drawRoundedFill(screen, labelRect.x, labelRect.y, labelRect.w, labelRect.h, 10, inputFill)
		drawRoundedFill(screen, labelRect.x+10, labelRect.y+7, 18, 18, 8, palette.Primary)
		drawUIText(screen, teamColorLabel(a.menu.soloColor), uiSmallFace, labelRect.x+40, labelRect.y+9, lightText)
		drawOverlayButton(screen, nextRect, ">", pointInRect(float64(cursorX), float64(cursorY), nextRect), false)
		help := "Use Left and Right or click the arrows to change your solo team color"
		drawUIText(screen, help, uiSmallFace, x+34, helpY, accentText)
	case menuOptionHost:
		drawUIText(screen, "Host Multiplayer", uiSmallFace, x+34, titleY, accentText)
		help := "Starts a local server, advertises it on your LAN, and joins from this client"
		drawUIText(screen, help, uiSmallFace, x+34, rowY+6, lightText)
	case menuOptionJoin:
		drawUIText(screen, "Join Multiplayer", uiSmallFace, x+34, titleY, accentText)
		drawRoundedFill(screen, x+34, rowY, width-68, 32, 10, inputFill)
		roomSummary := "No LAN rooms found yet"
		if len(a.menu.rooms) == 1 {
			roomSummary = "1 LAN room found nearby"
		}
		if len(a.menu.rooms) > 1 {
			roomSummary = fmt.Sprintf("%d LAN rooms found nearby", len(a.menu.rooms))
		}
		drawUIText(screen, roomSummary, uiSmallFace, x+48, rowY+8, lightText)
		help := "Press Enter to browse rooms, or click Join Multiplayer again"
		drawUIText(screen, help, uiSmallFace, x+34, helpY, accentText)
	}
}

func (a *App) drawRoomCard(screen *ebiten.Image, card rect, room discovery.Room, selected, hovered bool) {
	fill := color.RGBA{0x1a, 0x35, 0x56, 0xff}
	outline := color.RGBA{0x36, 0x63, 0x92, 0xff}
	titleColor := color.RGBA{0xf5, 0xfb, 0xff, 0xff}
	detailColor := color.RGBA{0xb8, 0xd8, 0xf5, 0xff}
	badgeFill := color.RGBA{0x0d, 0x1d, 0x31, 0xff}
	badgeText := color.RGBA{0xb6, 0xd9, 0xff, 0xff}
	if hovered {
		outline = color.RGBA{0x55, 0x92, 0xd1, 0xff}
	}
	if selected {
		fill = color.RGBA{0x25, 0x4d, 0x78, 0xff}
		outline = color.RGBA{0x6f, 0xb4, 0xff, 0xff}
		badgeFill = color.RGBA{0x12, 0x2a, 0x45, 0xff}
	}
	if !room.Joinable() {
		fill = color.RGBA{0x2a, 0x34, 0x41, 0xff}
		outline = color.RGBA{0x5e, 0x67, 0x74, 0xff}
		detailColor = color.RGBA{0xc9, 0xd2, 0xdd, 0xff}
		badgeFill = color.RGBA{0x1b, 0x22, 0x2c, 0xff}
		badgeText = color.RGBA{0xdd, 0xe4, 0xec, 0xff}
	}
	ebitenutil.DrawRect(screen, card.x+6, card.y+8, card.w, card.h, color.RGBA{0x02, 0x08, 0x11, 0x55})
	drawRoundedFill(screen, card.x, card.y, card.w, card.h, 20, fill)
	ebitenutil.DrawRect(screen, card.x, card.y, card.w, 2, outline)
	ebitenutil.DrawRect(screen, card.x, card.y+card.h-2, card.w, 2, outline)
	ebitenutil.DrawRect(screen, card.x, card.y, 2, card.h, outline)
	ebitenutil.DrawRect(screen, card.x+card.w-2, card.y, 2, card.h, outline)
	drawUIText(screen, "Room "+room.Code, uiBodyFace, card.x+24, card.y+22, titleColor)
	hostLabel := truncateLabel(room.Name, 24)
	drawUIText(screen, hostLabel, uiSmallFace, card.x+24, card.y+54, detailColor)
	badgeWidth := 112.0
	drawRoundedFill(screen, card.x+card.w-badgeWidth-18, card.y+18, badgeWidth, 34, 12, badgeFill)
	playersLabel := fmt.Sprintf("%d/%d players", room.Status.Players, room.Status.Capacity)
	if !room.Joinable() {
		playersLabel = "Room Full"
	}
	playersWidth, _ := measureUIText(playersLabel, uiSmallFace)
	drawUIText(screen, playersLabel, uiSmallFace, card.x+card.w-badgeWidth/2-playersWidth/2-18, card.y+28, badgeText)
}

func (a *App) startHostedRemote(listenAddr string) error {
	a.stopHostedServer()
	srv, err := server.Listen(listenAddr)
	if err != nil {
		return err
	}
	advertiser, err := discovery.NewAdvertiser(srv.Addr(), func() discovery.Status {
		return discovery.Status{Players: srv.PlayerCount(), Capacity: 2}
	})
	if err != nil {
		_ = srv.Close()
		return err
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve()
	}()
	joinAddr := localJoinAddress(srv.Addr())
	time.Sleep(150 * time.Millisecond)
	clientConn, err := netcode.Dial(joinAddr)
	if err != nil {
		_ = advertiser.Close()
		_ = srv.Close()
		return err
	}
	a.hostServer = srv
	a.hostServeErr = serveErr
	a.hostAdvertiser = advertiser
	a.remote = newRemoteGame(clientConn)
	a.solo = nil
	a.screen = appScreenRemote
	a.menu.status = fmt.Sprintf("Hosting room %s", advertiser.Room().Code)
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
	a.menu.status = "Connected to room"
	ebiten.SetWindowTitle(fmt.Sprintf("Go Hockey - Online %s", strings.ToUpper(string(a.remote.localTeam))))
	return nil
}

func (a *App) returnToMenu(status string) {
	if a.remote != nil && a.remote.client != nil {
		_ = a.remote.client.Close()
	}
	a.remote = nil
	a.solo = nil
	a.stopHostedServer()
	a.screen = appScreenMenu
	if status != "" {
		a.menu.status = status
	}
	ebiten.SetWindowTitle("Go Hockey")
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

func launcherFooterRect() rect {
	return rect{x: sim.CenterX - 290.0, y: 536.0, w: 580.0, h: 112.0}
}

func launcherSoloColorPrevRect() rect {
	footer := launcherFooterRect()
	return rect{x: footer.x + 34, y: footer.y + 64, w: 34, h: 30}
}

func launcherSoloColorLabelRect() rect {
	footer := launcherFooterRect()
	return rect{x: footer.x + 78, y: footer.y + 64, w: 136, h: 30}
}

func launcherSoloColorNextRect() rect {
	footer := launcherFooterRect()
	return rect{x: footer.x + 224, y: footer.y + 64, w: 34, h: 30}
}

func menuOptionRect(index int) rect {
	cardX := sim.CenterX - 250.0
	cardY := 176.0
	cardWidth := 500.0
	cardHeight := 98.0
	gap := 22.0
	return rect{
		x: cardX,
		y: cardY + float64(index)*(cardHeight+gap),
		w: cardWidth,
		h: cardHeight,
	}
}

func (a *App) menuOptionAtCursor() (menuOption, bool) {
	cursorX, cursorY := ebiten.CursorPosition()
	for index := 0; index < 3; index++ {
		if pointInRect(float64(cursorX), float64(cursorY), menuOptionRect(index)) {
			return menuOption(index), true
		}
	}
	return 0, false
}

type joinRoomCard struct {
	index int
	rect  rect
}

func (a *App) joinRoomCards() []joinRoomCard {
	if len(a.menu.rooms) == 0 {
		return nil
	}
	const visibleCount = 4
	start := 0
	if a.menu.roomCursor >= visibleCount {
		start = a.menu.roomCursor - visibleCount + 1
	}
	if start+visibleCount > len(a.menu.rooms) {
		start = len(a.menu.rooms) - visibleCount
	}
	if start < 0 {
		start = 0
	}
	cards := []joinRoomCard{}
	baseX := sim.CenterX - 280.0
	baseY := 226.0
	cardWidth := 560.0
	cardHeight := 72.0
	gap := 16.0
	end := start + visibleCount
	if end > len(a.menu.rooms) {
		end = len(a.menu.rooms)
	}
	for index := start; index < end; index++ {
		cards = append(cards, joinRoomCard{
			index: index,
			rect: rect{
				x: baseX,
				y: baseY + float64(index-start)*(cardHeight+gap),
				w: cardWidth,
				h: cardHeight,
			},
		})
	}
	return cards
}

func (a *App) joinRoomAtCursor() (int, bool) {
	cursorX, cursorY := ebiten.CursorPosition()
	for _, card := range a.joinRoomCards() {
		if pointInRect(float64(cursorX), float64(cursorY), card.rect) {
			return card.index, true
		}
	}
	return 0, false
}

func pointInRect(x, y float64, area rect) bool {
	return x >= area.x && x <= area.x+area.w && y >= area.y && y <= area.y+area.h
}

func roomKey(room discovery.Room) string {
	return room.Code + "|" + room.Addr
}

func truncateLabel(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
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

func localJoinAddress(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	if strings.HasPrefix(addr, "[::]:") {
		return "127.0.0.1:" + strings.TrimPrefix(addr, "[::]:")
	}
	return addr
}
