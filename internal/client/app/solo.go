package app

import (
	"fmt"

	clientinput "hockeyv2/internal/client/input"
	"hockeyv2/internal/client/render"
	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type SoloGame struct {
	state      sim.GameState
	menu       matchMenuState
	action     matchMenuAction
	sounds     *soundboard
	standalone bool
}

func NewSoloGame() *SoloGame {
	return NewSoloGameWithColors(sim.TeamColorBlue, sim.TeamColorRed)
}

func NewSoloGameWithColors(homeColor, awayColor sim.TeamColor) *SoloGame {
	state := sim.NewGameState()
	state.UseMenus = true
	state.HomeColor = homeColor
	state.AwayColor = awayColor
	return &SoloGame{state: state, sounds: defaultSoundboard()}
}

func (g *SoloGame) ConsumeAction() matchMenuAction {
	action := g.action
	g.action = matchMenuActionNone
	return action
}

func (g *SoloGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(sim.WindowWidth), int(sim.WindowHeight)
}

func (g *SoloGame) Update() error {
	if g.sounds != nil {
		g.sounds.PlayArenaAmbience()
	}
	g.syncMenuState()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.menu.Mode == matchMenuModePause {
			g.menu.Close()
		} else if g.menu.Mode == matchMenuModeHidden && !g.state.GameOver && g.state.Phase == sim.MatchPhasePlaying {
			g.menu.Open(matchMenuModePause)
		}
	}
	if g.menu.Visible() {
		g.updateMatchMenu()
		if g.standalone && g.action != matchMenuActionNone {
			if g.sounds != nil {
				g.sounds.StopArenaAmbience()
			}
			return ebiten.Termination
		}
		return nil
	}

	input := sim.InputFrame{
		Team:   sim.TeamHome,
		Move:   clientinput.MovementVector(),
		Shoot:  inpututil.IsKeyJustPressed(ebiten.KeySpace),
		Pass:   inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight),
		Switch: inpututil.IsKeyJustPressed(ebiten.KeyTab),
	}
	previousState := g.state
	sim.Step(&g.state, []sim.InputFrame{input})
	playMatchStateSounds(g.sounds, previousState, g.state)
	if g.standalone && g.action != matchMenuActionNone {
		if g.sounds != nil {
			g.sounds.StopArenaAmbience()
		}
		return ebiten.Termination
	}
	return nil
}

func (g *SoloGame) Draw(screen *ebiten.Image) {
	render.DrawMatch(screen, g.state, sim.TeamHome)
	render.DrawSoloHUD(screen, g.state, g.soloStatus())
	if g.menu.Visible() {
		title, subtitle, footer, entries := g.matchMenuContent()
		ui.DrawModalMenu(screen, title, subtitle, footer, entries, g.menu.Selected)
	}
}

func (g *SoloGame) syncMenuState() {
	if g.state.GameOver {
		if g.menu.Mode != matchMenuModePostgame {
			g.menu.Open(matchMenuModePostgame)
		}
		return
	}
	if g.state.Phase == sim.MatchPhaseIntermission {
		if g.menu.Mode != matchMenuModeIntermission {
			g.menu.Open(matchMenuModeIntermission)
		}
		return
	}
	if g.menu.Mode == matchMenuModePostgame || g.menu.Mode == matchMenuModeIntermission {
		g.menu.Close()
	}
}

func (g *SoloGame) restartMatch() {
	homeColor := g.state.HomeColor
	awayColor := g.state.AwayColor
	g.state = sim.NewGameState()
	g.state.UseMenus = true
	g.state.HomeColor = homeColor
	g.state.AwayColor = awayColor
	g.menu.Close()
}

func (g *SoloGame) continueIntermission() {
	g.state.Phase = sim.MatchPhasePlaying
	g.state.PhaseTicks = 0
	g.state.HomeReady = false
	g.state.AwayReady = false
	g.menu.Close()
}

func (g *SoloGame) updateMatchMenu() {
	entries := g.matchMenuEntries()
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
			g.restartMatch()
		case 2:
			g.action = matchMenuActionQuit
		}
	case matchMenuModeIntermission:
		switch activatedIndex {
		case 0:
			g.continueIntermission()
		case 1:
			g.restartMatch()
		case 2:
			g.action = matchMenuActionQuit
		}
	case matchMenuModePostgame:
		switch activatedIndex {
		case 0:
			g.restartMatch()
		case 1:
			g.action = matchMenuActionQuit
		}
	}
}

func (g *SoloGame) matchMenuContent() (string, string, string, []ui.MenuEntry) {
	switch g.menu.Mode {
	case matchMenuModePause:
		return "Pause Menu", "Match paused. Choose what you want to do next.", "Enter selects. Esc resumes the match.", []ui.MenuEntry{{Label: g.resumeLabel()}, {Label: g.restartLabel()}, {Label: g.quitLabel()}}
	case matchMenuModeIntermission:
		return g.intermissionTitle(), g.intermissionSubtitle(), "Enter continues to the next faceoff.", []ui.MenuEntry{{Label: g.continueLabel()}, {Label: g.restartLabel()}, {Label: g.quitLabel()}}
	case matchMenuModePostgame:
		return g.postgameTitle(), "The match is over.", "Enter selects an option.", []ui.MenuEntry{{Label: g.playAgainLabel()}, {Label: g.quitLabel()}}
	default:
		return "", "", "", nil
	}
}

func (g *SoloGame) matchMenuEntries() []ui.MenuEntry {
	_, _, _, entries := g.matchMenuContent()
	return entries
}

func (g *SoloGame) soloStatus() string {
	status := "Solo mode  WASD move  Shift pass  Space shoot/check  Tab switch  Esc menu"
	if g.menu.Mode == matchMenuModePause {
		status = "Paused  Choose Resume, Restart Match, or Quit"
	}
	if g.menu.Mode == matchMenuModeIntermission {
		status = "Intermission  Choose Continue, Restart Match, or Quit"
	}
	if g.menu.Mode == matchMenuModePostgame {
		status = "Game over  Choose Play Again or Quit"
	}
	return status
}

func (g *SoloGame) postgameTitle() string {
	switch {
	case g.state.Score.Home > g.state.Score.Away:
		return "You Win"
	case g.state.Score.Home < g.state.Score.Away:
		return "You Lose"
	default:
		return "Game Over"
	}
}

func (g *SoloGame) intermissionTitle() string {
	period := g.state.LastIntermissionStats.Period
	if period <= 0 && g.state.Period > 1 {
		period = g.state.Period - 1
	}
	if period <= 0 {
		return "Intermission"
	}
	return fmt.Sprintf("End of Period %d", period)
}

func (g *SoloGame) intermissionSubtitle() string {
	stats := g.state.LastIntermissionStats
	if stats.Period <= 0 {
		return "Take a breather, then continue when you are ready."
	}
	return fmt.Sprintf("Home %d shots / %d goals   Away %d shots / %d goals", stats.Home.ShotsOnGoal, stats.Home.Goals, stats.Away.ShotsOnGoal, stats.Away.Goals)
}

func (g *SoloGame) resumeLabel() string {
	return "Resume"
}

func (g *SoloGame) continueLabel() string {
	return "Continue"
}

func (g *SoloGame) restartLabel() string {
	return "Restart Match"
}

func (g *SoloGame) playAgainLabel() string {
	return "Play Again"
}

func (g *SoloGame) quitLabel() string {
	if g.standalone {
		return "Quit Game"
	}
	return "Quit to Launcher"
}
