package client

import (
	"fmt"
	"strings"

	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type RemoteGame struct {
	SoloGame
	client       *netcode.Client
	localTeam    sim.Team
	serverAddr   string
	disconnected string
}

func RunRemote(addr string) error {
	game, err := NewRemoteGame(addr)
	if err != nil {
		return err
	}
	defer game.Close()

	teamLabel := strings.ToUpper(string(game.localTeam))
	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle(fmt.Sprintf("Hockey 26 v2 - Online %s", teamLabel))
	ebiten.SetTPS(sim.TickRate)
	return ebiten.RunGame(game)
}

func NewRemoteGame(addr string) (*RemoteGame, error) {
	clientConn, err := netcode.Dial(addr)
	if err != nil {
		return nil, err
	}
	return newRemoteGame(clientConn, addr), nil
}

func newRemoteGame(clientConn *netcode.Client, addr string) *RemoteGame {
	return &RemoteGame{
		SoloGame:   SoloGame{state: sim.NewGameState()},
		client:     clientConn,
		localTeam:  clientConn.Team(),
		serverAddr: addr,
	}
}

func (g *RemoteGame) Close() error {
	if g.client == nil {
		return nil
	}
	return g.client.Close()
}

func (g *RemoteGame) Update() error {
	for {
		select {
		case snapshot := <-g.client.Snapshots():
			g.state = snapshot
		default:
			goto snapshotsDone
		}
	}

snapshotsDone:
	select {
	case err := <-g.client.Errors():
		g.disconnected = err.Error()
	default:
	}

	if g.disconnected != "" {
		return nil
	}

	input := g.currentInput()
	if err := g.client.SendInput(input); err != nil {
		g.disconnected = err.Error()
	}
	return nil
}

func (g *RemoteGame) currentInput() sim.InputFrame {
	if g.state.Phase != sim.MatchPhasePlaying {
		return sim.InputFrame{
			Team:      g.localTeam,
			Tick:      g.state.Tick + 1,
			Ready:     inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter),
			ColorPrev: inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft),
			ColorNext: inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight),
		}
	}
	return sim.InputFrame{
		Team:   g.localTeam,
		Tick:   g.state.Tick + 1,
		Move:   movementVector(),
		Shoot:  inpututil.IsKeyJustPressed(ebiten.KeySpace),
		Pass:   inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight),
		Switch: inpututil.IsKeyJustPressed(ebiten.KeyTab),
	}
}

func (g *RemoteGame) Draw(screen *ebiten.Image) {
	screen.Fill(colorHUDBackground)
	g.drawMatch(screen, g.localTeam)
	if g.state.Phase != sim.MatchPhasePlaying && !g.state.GameOver {
		g.drawReadyOverlay(screen, g.localTeam, fmt.Sprintf("Connected to %s", g.serverAddr))
	}
	g.drawNetworkHUD(screen)
}

func (g *RemoteGame) drawNetworkHUD(screen *ebiten.Image) {
	minutes := g.state.ClockTicks / (sim.TickRate * 60)
	seconds := (g.state.ClockTicks / sim.TickRate) % 60
	periodLabel := fmt.Sprintf("P%d", g.state.Period)
	if g.state.InOvertime {
		periodLabel = "OT"
	}
	status := fmt.Sprintf("Online %s  %s  WASD move  Shift pass  Space shoot/check  Tab switch", strings.ToUpper(string(g.localTeam)), g.serverAddr)
	if g.state.Phase != sim.MatchPhasePlaying {
		status = "Menu controls: A/Left and D/Right change color  Space or Enter toggles ready"
	}
	if g.disconnected != "" {
		status = fmt.Sprintf("Disconnected: %s  Press Esc for the launcher", g.disconnected)
	}
	if g.state.GameOver {
		status = "Game over on server  Press Esc for the launcher"
	}
	ebitenutil.DebugPrintAt(screen, "Hockey 26 v2 Online", 20, 18)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d - %d", g.state.Score.Home, g.state.Score.Away), int(sim.CenterX)-24, 20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s %02d:%02d", periodLabel, minutes, seconds), 20, 42)
	ebitenutil.DebugPrintAt(screen, status, 20, int(sim.WindowHeight)-28)
}
