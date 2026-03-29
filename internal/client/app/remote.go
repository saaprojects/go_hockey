package app

import (
	"fmt"
	"strings"

	clientinput "hockeyv2/internal/client/input"
	"hockeyv2/internal/client/render"
	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type RemoteGame struct {
	client             *netcode.Client
	localTeam          sim.Team
	state              sim.GameState
	menu               matchMenuState
	action             matchMenuAction
	sounds             *soundboard
	standalone         bool
	disconnected       string
	pendingRematchVote bool
	roomCode           string
	roomName           string
	host               bool
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
		client:    clientConn,
		localTeam: clientConn.Team(),
		state:     sim.NewGameState(),
		sounds:    defaultSoundboard(),
		roomCode:  clientConn.RoomCode(),
		roomName:  clientConn.RoomName(),
		host:      clientConn.IsHost(),
	}
}

func (g *RemoteGame) Close() error {
	if g.sounds != nil {
		g.sounds.StopArenaAmbience()
	}
	if g.client == nil {
		return nil
	}
	return g.client.Close()
}

func (g *RemoteGame) ConsumeAction() matchMenuAction {
	action := g.action
	g.action = matchMenuActionNone
	return action
}

func (g *RemoteGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(sim.WindowWidth), int(sim.WindowHeight)
}

func (g *RemoteGame) Update() error {
	if g.sounds != nil {
		g.sounds.PlayArenaAmbience()
	}
	for {
		select {
		case snapshot := <-g.client.Snapshots():
			previousState := g.state
			g.state = snapshot
			playMatchStateSounds(g.sounds, previousState, g.state)
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

	g.syncMenuState()
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
		if err := g.client.SendInput(g.currentInput()); err != nil {
			g.disconnected = "Disconnected from server"
			g.syncMenuState()
		}
	}
	if g.standalone && g.action != matchMenuActionNone {
		return ebiten.Termination
	}
	return nil
}

func (g *RemoteGame) Draw(screen *ebiten.Image) {
	render.DrawMatch(screen, g.state, g.localTeam)
	if g.state.Phase != sim.MatchPhasePlaying && !g.state.GameOver {
		overlayStatus := "Connected to online match"
		if g.roomCode != "" {
			overlayStatus = fmt.Sprintf("Room %s", g.roomCode)
			if g.host {
				overlayStatus += "  Host"
			}
		}
		render.DrawReadyOverlay(screen, g.state, g.localTeam, overlayStatus)
	}
	hudTitle := fmt.Sprintf("Go Hockey Online %s", strings.ToUpper(string(g.localTeam)))
	if g.roomCode != "" {
		hudTitle = fmt.Sprintf("Room %s %s", g.roomCode, strings.ToUpper(string(g.localTeam)))
		if g.host {
			hudTitle = "Host " + hudTitle
		}
	}
	render.DrawNetworkHUD(screen, g.state, hudTitle, g.networkStatus())
	if g.menu.Visible() {
		title, subtitle, footer := g.remoteMenuText()
		ui.DrawModalMenu(screen, title, subtitle, footer, g.remoteMenuEntries(), g.menu.Selected)
	}
}

func (g *RemoteGame) syncMenuState() {
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
	selected, activatedIndex, activated := clientinput.UpdateSelectableMenu(g.menu.Selected, entries, func(index int) ui.Rect {
		return ui.ModalMenuOptionRect(index, len(entries))
	})
	g.menu.Selected = selected
	if !activated {
		return
	}
	switch g.menu.Mode {
	case matchMenuModePause:
		switch activatedIndex {
		case 0:
			g.menu.Close()
		case 1:
			g.action = matchMenuActionQuit
		case 2:
			g.action = g.roomMenuAction()
		}
	case matchMenuModePostgame:
		switch activatedIndex {
		case 0:
			if !g.localTeamReady() {
				g.pendingRematchVote = true
			}
		case 1:
			g.action = matchMenuActionQuit
		case 2:
			g.action = g.roomMenuAction()
		}
	case matchMenuModeDisconnected:
		switch activatedIndex {
		case 0:
			g.action = matchMenuActionQuit
		case 1:
			g.action = g.roomMenuAction()
		}
	}
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
		action := clientinput.ReadyOverlayMouseAction(
			render.ReadyOverlayColorPrevRect(g.localTeam),
			render.ReadyOverlayColorNextRect(g.localTeam),
			render.ReadyOverlayReadyRect(g.localTeam),
		)
		input.Ready = inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || action.Ready
		input.ColorPrev = inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || action.ColorPrev
		input.ColorNext = inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || action.ColorNext
		return input
	}
	input.Move = clientinput.MovementVector()
	input.Shoot = inpututil.IsKeyJustPressed(ebiten.KeySpace)
	input.Pass = inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight)
	input.Switch = inpututil.IsKeyJustPressed(ebiten.KeyTab)
	return input
}

func (g *RemoteGame) remoteMenuEntries() []ui.MenuEntry {
	roomEntry := ui.MenuEntry{Label: "Room Menu"}
	if g.standalone {
		roomEntry = ui.MenuEntry{Label: "Quit Match"}
	}
	switch g.menu.Mode {
	case matchMenuModePause:
		return []ui.MenuEntry{{Label: "Resume"}, {Label: "Quit Match"}, roomEntry}
	case matchMenuModePostgame:
		playAgain := ui.MenuEntry{Label: "Play Again"}
		if g.localTeamReady() || g.pendingRematchVote {
			playAgain.Label = "Waiting for Other Player"
			playAgain.Disabled = true
		}
		return []ui.MenuEntry{playAgain, {Label: "Quit Match"}, roomEntry}
	case matchMenuModeDisconnected:
		return []ui.MenuEntry{{Label: "Quit Match"}, roomEntry}
	default:
		return nil
	}
}

func (g *RemoteGame) remoteMenuText() (string, string, string) {
	roomHint := ""
	if g.roomCode != "" {
		roomHint = fmt.Sprintf(" Room code: %s.", g.roomCode)
	}
	hostHint := ""
	if g.host {
		hostHint = " You created this room."
	}
	switch g.menu.Mode {
	case matchMenuModePause:
		subtitle := "Your player will idle while this menu is open."
		if g.roomName != "" {
			subtitle = fmt.Sprintf("%s is still live while this menu is open.", g.roomName)
		}
		return "Match Menu", subtitle + roomHint + hostHint, "Enter selects. Esc returns to the match."
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
		return title, subtitle, "Both players must choose Play Again to restart." + roomHint
	case matchMenuModeDisconnected:
		return "Connection Lost", "The match is no longer connected to the server.", "Choose Quit Match or Room Menu."
	default:
		return "", "", ""
	}
}

func (g *RemoteGame) networkStatus() string {
	roomPrefix := ""
	if g.roomCode != "" {
		roomPrefix = fmt.Sprintf("Room %s  ", g.roomCode)
		if g.host {
			roomPrefix = fmt.Sprintf("Host %s", roomPrefix)
		}
	}
	status := fmt.Sprintf("%sOnline %s  WASD move  Shift pass  Space shoot/check  Tab switch  Esc menu", roomPrefix, strings.ToUpper(string(g.localTeam)))
	if g.state.Phase != sim.MatchPhasePlaying && !g.state.GameOver {
		status = "Menu controls: A/Left and D/Right or click arrows change color  Space/Enter or click Ready toggles ready"
		if g.roomCode != "" {
			status = fmt.Sprintf("Room %s  %s", g.roomCode, status)
			if g.host {
				status = "Host " + status
			}
		}
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
	return status
}

func (g *RemoteGame) roomMenuAction() matchMenuAction {
	if g.standalone {
		return matchMenuActionQuit
	}
	return matchMenuActionRoomMenu
}

func (g *RemoteGame) localTeamReady() bool {
	if g.localTeam == sim.TeamHome {
		return g.state.HomeReady
	}
	return g.state.AwayReady
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
