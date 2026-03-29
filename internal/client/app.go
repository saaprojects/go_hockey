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

const defaultJoinAddress = "127.0.0.1:4242"

type App struct {
	screen       appScreen
	menu         launchMenu
	solo         *SoloGame
	remote       *RemoteGame
	hostServer   *server.Server
	hostServeErr chan error
}

type launchMenu struct {
	selected menuOption
	joinAddr string
	status   string
}

func RunApp() error {
	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle("Hockey 26 v2")
	ebiten.SetTPS(sim.TickRate)
	return ebiten.RunGame(NewApp())
}

func NewApp() *App {
	return &App{
		screen: appScreenMenu,
		menu: launchMenu{
			selected: menuOptionSolo,
			joinAddr: defaultJoinAddress,
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
				a.returnToMenu(fmt.Sprintf("Host stopped: %v", err))
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
		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			a.menu.joinAddr = defaultJoinAddress
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch a.menu.selected {
		case menuOptionSolo:
			a.solo = NewSoloGame()
			a.screen = appScreenSolo
			ebiten.SetWindowTitle("Hockey 26 v2 - Solo")
		case menuOptionHost:
			if err := a.startHostedRemote(":4242"); err != nil {
				a.menu.status = err.Error()
			}
		case menuOptionJoin:
			addr := strings.TrimSpace(a.menu.joinAddr)
			if addr == "" {
				a.menu.status = "Enter a server address like 127.0.0.1:4242"
				return nil
			}
			if err := a.startRemote(addr); err != nil {
				a.menu.status = err.Error()
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
	footerX := sim.CenterX - 330.0
	footerY := 494.0
	footerWidth := 660.0
	footerHeight := 126.0

	topFill := color.RGBA{0xf8, 0xfb, 0xff, 0xf4}
	footerFill := color.RGBA{0x10, 0x22, 0x39, 0xdd}
	mutedText := color.RGBA{0x5b, 0x6c, 0x80, 0xff}
	lightText := color.RGBA{0xee, 0xf5, 0xff, 0xff}
	accentText := color.RGBA{0x9e, 0xcf, 0xff, 0xff}

	ebitenutil.DrawRect(screen, topPanelX+8, topPanelY+10, topPanelWidth, topPanelHeight, colorPanelShadow)
	drawRoundedFill(screen, topPanelX, topPanelY, topPanelWidth, topPanelHeight, 24, topFill)
	title := "Hockey 26 v2"
	titleWidth, _ := measureUIText(title, uiTitleFace)
	drawUIText(screen, title, uiTitleFace, sim.CenterX-titleWidth/2, 60, colorTextDark)
	subtitle := "Choose solo, host, or join from this same client"
	subtitleWidth, _ := measureUIText(subtitle, uiBodyFace)
	drawUIText(screen, subtitle, uiBodyFace, sim.CenterX-subtitleWidth/2, 96, colorTextDark)
	controls := "Enter or Space starts, Up or Down chooses, Esc returns here from a match"
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
		fmt.Sprintf("Connect to another host at %s", a.menu.joinAddr),
	}
	for index, label := range labels {
		y := cardY + float64(index)*(cardHeight+gap)
		a.drawMenuOptionCard(screen, cardX, y, cardWidth, cardHeight, label, details[index], a.menu.selected == menuOption(index))
	}

	ebitenutil.DrawRect(screen, footerX+8, footerY+10, footerWidth, footerHeight, color.RGBA{0x03, 0x0b, 0x14, 0x55})
	drawRoundedFill(screen, footerX, footerY, footerWidth, footerHeight, 24, footerFill)
	colorsLine := "Multiplayer colors: Black  |  Orange  |  Green  |  Blue  |  Red"
	colorsWidth, _ := measureUIText(colorsLine, uiBodyFace)
	drawUIText(screen, colorsLine, uiBodyFace, sim.CenterX-colorsWidth/2, 522, lightText)
	joinHelp := "When Join Multiplayer is selected, type to edit the address. Backspace deletes. Tab resets to 127.0.0.1:4242."
	joinHelpWidth, _ := measureUIText(joinHelp, uiSmallFace)
	drawUIText(screen, joinHelp, uiSmallFace, sim.CenterX-joinHelpWidth/2, 556, accentText)
	if a.menu.status != "" {
		statusWidth, _ := measureUIText(a.menu.status, uiBodyFace)
		drawUIText(screen, a.menu.status, uiBodyFace, sim.CenterX-statusWidth/2, 592, lightText)
	}
}

func (a *App) drawMenuOptionCard(screen *ebiten.Image, x, y, width, height float64, label, detail string, selected bool) {
	cardFill := color.RGBA{0xf7, 0xfb, 0xff, 0xfa}
	stripe := color.RGBA{0x4e, 0x72, 0x97, 0xff}
	titleColor := colorTextDark
	detailColor := color.RGBA{0x4f, 0x60, 0x74, 0xff}
	shadow := colorPanelShadow
	if selected {
		cardFill = color.RGBA{0x16, 0x35, 0x58, 0xfa}
		stripe = color.RGBA{0x46, 0x9b, 0xff, 0xff}
		titleColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
		detailColor = color.RGBA{0xdd, 0xee, 0xff, 0xff}
		shadow = color.RGBA{0x02, 0x08, 0x11, 0x70}
	}
	ebitenutil.DrawRect(screen, x+8, y+10, width, height, shadow)
	drawRoundedFill(screen, x, y, width, height, 20, cardFill)
	ebitenutil.DrawRect(screen, x, y, 18, height, stripe)
	drawUIText(screen, label, uiBodyFace, x+36, y+24, titleColor)
	drawUIText(screen, detail, uiSmallFace, x+36, y+58, detailColor)
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
	a.remote = newRemoteGame(clientConn, joinAddr)
	a.solo = nil
	a.screen = appScreenRemote
	a.menu.status = fmt.Sprintf("Hosting on %s", srv.Addr())
	ebiten.SetWindowTitle("Hockey 26 v2 - Host Multiplayer")
	return nil
}

func (a *App) startRemote(addr string) error {
	a.stopHostedServer()
	clientConn, err := netcode.Dial(addr)
	if err != nil {
		return err
	}
	a.remote = newRemoteGame(clientConn, addr)
	a.solo = nil
	a.screen = appScreenRemote
	a.hostServeErr = nil
	a.menu.status = fmt.Sprintf("Joined %s", addr)
	ebiten.SetWindowTitle(fmt.Sprintf("Hockey 26 v2 - Online %s", strings.ToUpper(string(a.remote.localTeam))))
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
	ebiten.SetWindowTitle("Hockey 26 v2")
}

func (a *App) stopHostedServer() {
	if a.hostServer != nil {
		_ = a.hostServer.Close()
		a.hostServer = nil
	}
	a.hostServeErr = nil
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
