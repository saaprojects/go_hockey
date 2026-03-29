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
	client             *netcode.Client
	localTeam          sim.Team
	disconnected       string
	pendingRematchVote bool
}

func RunRemote(addr string) error {
	game, err := NewRemoteGame(addr)
	if err != nil {
		return err
	}
	defer game.Close()

	teamLabel := strings.ToUpper(string(game.localTeam))
	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle(fmt.Sprintf("Go Hockey - Online %s", teamLabel))
	ebiten.SetTPS(sim.TickRate)
	return ebiten.RunGame(game)
}

func NewRemoteGame(addr string) (*RemoteGame, error) {
	clientConn, err := netcode.Dial(addr)
	if err != nil {
		return nil, err
	}
	game := newRemoteGame(clientConn)
	game.standalone = true
	return game, nil
}

func newRemoteGame(clientConn *netcode.Client) *RemoteGame {
	return &RemoteGame{
		SoloGame:  SoloGame{state: sim.NewGameState()},
		client:    clientConn,
		localTeam: clientConn.Team(),
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
	case <-g.client.Errors():
		g.disconnected = "Disconnected from server"
	default:
	}

	g.syncRemoteMenuState()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.menu.Mode == matchMenuModePause {
			g.menu.Close()
		} else if g.menu.Mode == matchMenuModeHidden && g.disconnected == "" && !g.state.GameOver {
			g.menu.Open(matchMenuModePause)
		}
	}
	if g.menu.Visible() {
		g.updateRemoteMenu()
	}
	if g.disconnected == "" {
		input := g.currentInput()
		if err := g.client.SendInput(input); err != nil {
			g.disconnected = "Disconnected from server"
			g.syncRemoteMenuState()
		}
	}
	if g.standalone && g.action != matchMenuActionNone {
		return ebiten.Termination
	}
	return nil
}

func (g *RemoteGame) syncRemoteMenuState() {
	switch {
	case g.disconnected != "":
		if g.menu.Mode != matchMenuModeDisconnected {
			g.menu.Open(matchMenuModeDisconnected)
		}
	case g.state.GameOver:
		if g.menu.Mode != matchMenuModePostgame {
			g.menu.Open(matchMenuModePostgame)
		}
	default:
		if g.menu.Mode == matchMenuModePostgame || g.menu.Mode == matchMenuModeDisconnected {
			g.menu.Close()
		}
	}
}

func (g *RemoteGame) updateRemoteMenu() {
	entries := g.remoteMenuEntries()
	if len(entries) == 0 {
		return
	}
	if choice, activated := updateMatchMenuSelection(&g.menu, entries); activated {
		switch g.menu.Mode {
		case matchMenuModePause:
			switch choice {
			case 0:
				g.menu.Close()
			case 1:
				g.action = matchMenuActionQuit
			case 2:
				g.action = matchMenuActionRoomMenu
			}
		case matchMenuModePostgame:
			switch choice {
			case 0:
				if !g.localTeamReady() {
					g.pendingRematchVote = true
				}
			case 1:
				g.action = matchMenuActionQuit
			case 2:
				g.action = matchMenuActionRoomMenu
			}
		case matchMenuModeDisconnected:
			switch choice {
			case 0:
				g.action = matchMenuActionQuit
			case 1:
				g.action = matchMenuActionRoomMenu
			}
		}
	}
}

func (g *RemoteGame) remoteMenuEntries() []matchMenuEntry {
	switch g.menu.Mode {
	case matchMenuModePause:
		return []matchMenuEntry{{Label: "Resume"}, {Label: "Quit Match"}, {Label: "Room Menu"}}
	case matchMenuModePostgame:
		playAgain := matchMenuEntry{Label: "Play Again"}
		if g.localTeamReady() || g.pendingRematchVote {
			playAgain.Label = "Waiting for Other Player"
			playAgain.Disabled = true
		}
		return []matchMenuEntry{playAgain, {Label: "Quit Match"}, {Label: "Room Menu"}}
	case matchMenuModeDisconnected:
		return []matchMenuEntry{{Label: "Quit Match"}, {Label: "Room Menu"}}
	default:
		return nil
	}
}

func (g *RemoteGame) localTeamReady() bool {
	if g.localTeam == sim.TeamHome {
		return g.state.HomeReady
	}
	return g.state.AwayReady
}

func (g *RemoteGame) currentInput() sim.InputFrame {
	input := sim.InputFrame{Team: g.localTeam, Tick: g.state.Tick + 1}
	if g.state.GameOver {
		if g.pendingRematchVote {
			input.Ready = true
			g.pendingRematchVote = false
		}
		return input
	}
	if g.menu.Visible() {
		return input
	}
	if g.state.Phase != sim.MatchPhasePlaying {
		mouseAction := readyOverlayMouseAction(g.localTeam)
		input.Ready = inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || mouseAction.ready
		input.ColorPrev = inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || mouseAction.colorPrev
		input.ColorNext = inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || mouseAction.colorNext
		return input
	}
	input.Move = movementVector()
	input.Shoot = inpututil.IsKeyJustPressed(ebiten.KeySpace)
	input.Pass = inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight)
	input.Switch = inpututil.IsKeyJustPressed(ebiten.KeyTab)
	return input
}

func (g *RemoteGame) Draw(screen *ebiten.Image) {
	screen.Fill(colorHUDBackground)
	g.drawMatch(screen, g.localTeam)
	if g.state.Phase != sim.MatchPhasePlaying && !g.state.GameOver {
		g.drawReadyOverlay(screen, g.localTeam, "Connected to online match")
	}
	g.drawNetworkHUD(screen)
	if g.menu.Visible() {
		title, subtitle, footer := g.remoteMenuText()
		drawMatchMenuOverlay(screen, title, subtitle, footer, g.remoteMenuEntries(), g.menu.Selected)
	}
}

func (g *RemoteGame) remoteMenuText() (string, string, string) {
	switch g.menu.Mode {
	case matchMenuModePause:
		return "Match Menu", "Your player will idle while this menu is open.", "Enter selects. Esc returns to the match."
	case matchMenuModePostgame:
		title := "Game Over"
		opponent := g.opponentTeam()
		if g.scoreFor(g.localTeam) > g.scoreFor(opponent) {
			title = "You Win"
		} else if g.scoreFor(g.localTeam) < g.scoreFor(opponent) {
			title = "You Lose"
		}
		subtitle := "Choose Play Again to vote for a rematch."
		if g.localTeamReady() {
			subtitle = "Rematch requested. Waiting for the other player."
		}
		return title, subtitle, "Both players must choose Play Again to restart."
	case matchMenuModeDisconnected:
		return "Connection Lost", "The match is no longer connected to the server.", "Choose Quit Match or Room Menu."
	default:
		return "", "", ""
	}
}

func (g *RemoteGame) scoreFor(team sim.Team) int {
	if team == sim.TeamHome {
		return g.state.Score.Home
	}
	return g.state.Score.Away
}

func (g *RemoteGame) opponentTeam() sim.Team {
	if g.localTeam == sim.TeamHome {
		return sim.TeamAway
	}
	return sim.TeamHome
}

func (g *RemoteGame) drawNetworkHUD(screen *ebiten.Image) {
	minutes := g.state.ClockTicks / (sim.TickRate * 60)
	seconds := (g.state.ClockTicks / sim.TickRate) % 60
	periodLabel := fmt.Sprintf("P%d", g.state.Period)
	if g.state.InOvertime {
		periodLabel = "OT"
	}
	status := fmt.Sprintf("Online %s  WASD move  Shift pass  Space shoot/check  Tab switch  Esc menu", strings.ToUpper(string(g.localTeam)))
	if g.state.Phase != sim.MatchPhasePlaying && !g.state.GameOver {
		status = "Menu controls: A/Left and D/Right or click arrows change color  Space/Enter or click Ready toggles ready"
	}
	if g.menu.Mode == matchMenuModePause {
		status = "Match menu open  Choose Resume, Quit Match, or Room Menu"
	}
	if g.menu.Mode == matchMenuModePostgame {
		status = "Game over  Choose Play Again, Quit Match, or Room Menu"
	}
	if g.menu.Mode == matchMenuModeDisconnected {
		status = "Disconnected from server  Choose Quit Match or Room Menu"
	}
	ebitenutil.DebugPrintAt(screen, "Go Hockey Online", 20, 18)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d - %d", g.state.Score.Home, g.state.Score.Away), int(sim.CenterX)-24, 20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s %02d:%02d", periodLabel, minutes, seconds), 20, 42)
	ebitenutil.DebugPrintAt(screen, status, 20, int(sim.WindowHeight)-28)
}
