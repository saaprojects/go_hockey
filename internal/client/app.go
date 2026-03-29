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

	ebitenutil.DebugPrintAt(screen, "Hockey 26 v2", int(sim.CenterX)-52, 54)
	ebitenutil.DebugPrintAt(screen, "Choose a mode from the same client", int(sim.CenterX)-118, 80)
	ebitenutil.DebugPrintAt(screen, "Enter/Space start  Up/Down choose  Esc returns here from a match", int(sim.CenterX)-180, 104)

	cardX := sim.CenterX - 210
	cardY := 180.0
	cardWidth := 420.0
	cardHeight := 78.0
	gap := 20.0
	labels := []string{"Solo Game", "Host Multiplayer", "Join Multiplayer"}
	details := []string{
		"Play locally against AI in the same client.",
		"Start a local server and join it from this window.",
		fmt.Sprintf("Connect to another host. Address: %s", a.menu.joinAddr),
	}
	for index, label := range labels {
		y := cardY + float64(index)*(cardHeight+gap)
		selected := a.menu.selected == menuOption(index)
		fill := colorPanel
		stripe := color.RGBA{0x4a, 0x6c, 0x8f, 0xff}
		if selected {
			fill = color.RGBA{0xff, 0xff, 0xff, 0xf8}
			stripe = color.RGBA{0x1f, 0x7a, 0xe0, 0xff}
		}
		ebitenutil.DrawRect(screen, cardX+6, y+8, cardWidth, cardHeight, colorPanelShadow)
		drawRoundedFill(screen, cardX, y, cardWidth, cardHeight, 18, fill)
		ebitenutil.DrawRect(screen, cardX, y, 14, cardHeight, stripe)
		ebitenutil.DebugPrintAt(screen, label, int(cardX)+30, int(y)+18)
		ebitenutil.DebugPrintAt(screen, details[index], int(cardX)+30, int(y)+42)
	}

	paletteNames := []string{"Black", "Orange", "Green", "Blue", "Red"}
	ebitenutil.DebugPrintAt(screen, "Multiplayer team colors:", int(sim.CenterX)-90, 498)
	ebitenutil.DebugPrintAt(screen, strings.Join(paletteNames, "  |  "), int(sim.CenterX)-112, 522)
	if a.menu.selected == menuOptionJoin {
		ebitenutil.DebugPrintAt(screen, "Type to edit the join address. Backspace deletes. Tab resets to 127.0.0.1:4242.", int(sim.CenterX)-230, 556)
	}
	if a.menu.status != "" {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", a.menu.status), int(sim.CenterX)-160, 592)
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
