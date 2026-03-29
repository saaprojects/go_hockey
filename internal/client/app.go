package client

import (
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode"

	"hockeyv2/internal/netcode"
	"hockeyv2/internal/server"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type appScreen int

type menuOption int

const (
	appScreenMenu appScreen = iota
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
	screen       appScreen
	menu         launchMenu
	solo         *SoloGame
	remote       *RemoteGame
	hostServer   *server.Server
	hostServeErr chan error
}

type launchMenu struct {
	selected  menuOption
	joinAddr  string
	soloColor sim.TeamColor
	status    string
}

func RunApp() error {
	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle("Go Hockey")
	ebiten.SetTPS(sim.TickRate)
	return ebiten.RunGame(NewApp())
}

func NewApp() *App {
	return &App{
		screen: appScreenMenu,
		menu: launchMenu{
			selected:  menuOptionSolo,
			joinAddr:  "",
			soloColor: sim.TeamColorBlue,
		},
	}
}

func (a *App) Update() error {
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
	default:
		a.drawMenu(screen)
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(sim.WindowWidth), int(sim.WindowHeight)
}

func (a *App) updateMenu() error {
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
	if a.menu.selected == menuOptionJoin {
		for _, r := range ebiten.AppendInputChars(nil) {
			if isJoinAddressRune(r) {
				a.menu.joinAddr += string(r)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			runes := []rune(a.menu.joinAddr)
			if len(runes) > 0 {
				a.menu.joinAddr = string(runes[:len(runes)-1])
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch a.menu.selected {
		case menuOptionSolo:
			a.solo = NewSoloGameWithColors(a.menu.soloColor, awayColorForSolo(a.menu.soloColor))
			a.screen = appScreenSolo
			ebiten.SetWindowTitle("Go Hockey - Solo")
		case menuOptionHost:
			if err := a.startHostedRemote(":4242"); err != nil {
				a.menu.status = "Unable to host local server"
			}
		case menuOptionJoin:
			addr := strings.TrimSpace(a.menu.joinAddr)
			if addr == "" {
				a.menu.status = "Enter a server address to connect"
				return nil
			}
			if err := a.startRemote(addr); err != nil {
				a.menu.status = "Unable to connect to server"
			}
		}
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
	subtitle := "Choose solo, host, or join from this same client"
	subtitleWidth, _ := measureUIText(subtitle, uiBodyFace)
	drawUIText(screen, subtitle, uiBodyFace, sim.CenterX-subtitleWidth/2, 96, colorTextDark)
	controls := "Enter starts, Up or Down chooses, Esc returns here from a match"
	controlsWidth, _ := measureUIText(controls, uiSmallFace)
	drawUIText(screen, controls, uiSmallFace, sim.CenterX-controlsWidth/2, 122, mutedText)

	cardX := sim.CenterX - 250.0
	cardY := 176.0
	cardWidth := 500.0
	cardHeight := 98.0
	gap := 22.0
	labels := []string{"Solo Game", "Host Multiplayer", "Join Multiplayer"}
	details := []string{
		"Play locally against AI in the same client.",
		"Start a local server and join it from this window.",
		"Connect to another host after entering a server address.",
	}
	for index, label := range labels {
		y := cardY + float64(index)*(cardHeight+gap)
		a.drawMenuOptionCard(screen, cardX, y, cardWidth, cardHeight, label, details[index], a.menu.selected == menuOption(index), cardFill, selectedFill)
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
		drawUIText(screen, "Solo team color", uiSmallFace, x+34, titleY, accentText)
		drawRoundedFill(screen, x+34, rowY, 26, 26, 10, palette.Primary)
		drawUIText(screen, teamColorLabel(a.menu.soloColor), uiBodyFace, x+72, rowY-2, lightText)
		help := "Use Left and Right to change your solo team color before starting"
		drawUIText(screen, help, uiSmallFace, x+34, helpY, accentText)
	case menuOptionHost:
		drawUIText(screen, "Host Multiplayer", uiSmallFace, x+34, titleY, accentText)
		help := "Starts a local server and joins it from this client without exposing an address in the UI"
		drawUIText(screen, help, uiSmallFace, x+34, rowY+6, lightText)
	case menuOptionJoin:
		drawUIText(screen, "Server address", uiSmallFace, x+34, titleY, accentText)
		drawRoundedFill(screen, x+34, rowY, width-68, 32, 10, inputFill)
		if a.menu.joinAddr != "" {
			drawUIText(screen, a.menu.joinAddr, uiSmallFace, x+48, rowY+8, lightText)
		}
		help := "Type the address you want to join. Backspace deletes."
		drawUIText(screen, help, uiSmallFace, x+34, helpY, accentText)
	}
}

func (a *App) startHostedRemote(listenAddr string) error {
	a.stopHostedServer()
	srv, err := server.Listen(listenAddr)
	if err != nil {
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
		_ = srv.Close()
		return err
	}
	a.hostServer = srv
	a.hostServeErr = serveErr
	a.remote = newRemoteGame(clientConn)
	a.solo = nil
	a.screen = appScreenRemote
	a.menu.status = "Hosting local server"
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
	a.menu.status = "Connected to server"
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
	if a.hostServer != nil {
		_ = a.hostServer.Close()
		a.hostServer = nil
	}
	a.hostServeErr = nil
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

func isJoinAddressRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	return strings.ContainsRune(".:-[]", r)
}
