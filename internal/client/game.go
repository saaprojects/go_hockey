package client

import (
	"fmt"
	"image/color"
	"math"

	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	colorHUDBackground = color.RGBA{0x0b, 0x1d, 0x34, 0xff}
	colorBoardOutline  = color.RGBA{0x97, 0xa8, 0xb6, 0xff}
	colorBoard         = color.RGBA{0xfd, 0xfd, 0xfd, 0xff}
	colorIce           = color.RGBA{0xea, 0xf7, 0xff, 0xff}
	colorCenterRed     = color.RGBA{0xd6, 0x34, 0x34, 0xff}
	colorBlueLine      = color.RGBA{0x38, 0x7b, 0xe3, 0xff}
	colorCrease        = color.RGBA{0xcb, 0xe7, 0xff, 0xff}
	colorCreaseLine    = color.RGBA{0x62, 0xa0, 0xe8, 0xff}
	colorNetFrame      = color.RGBA{0xdf, 0x3a, 0x3a, 0xff}
	colorNetMesh       = color.RGBA{0xd7, 0xe0, 0xe7, 0xff}
	colorPuck          = color.RGBA{0x11, 0x12, 0x14, 0xff}
	colorOverlay       = color.RGBA{0x08, 0x12, 0x20, 0xdd}
	colorPanel         = color.RGBA{0xf2, 0xf7, 0xfc, 0xf0}
	colorPanelShadow   = color.RGBA{0x05, 0x0c, 0x15, 0x55}
	colorTextDark      = color.RGBA{0x12, 0x1d, 0x2b, 0xff}
)

type SoloGame struct {
	state  sim.GameState
	paused bool
}

func RunSolo() error {
	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle("Hockey 26 v2 - Solo")
	ebiten.SetTPS(sim.TickRate)
	return ebiten.RunGame(NewSoloGame())
}

func NewSoloGame() *SoloGame {
	return &SoloGame{state: sim.NewGameState()}
}

func (g *SoloGame) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.state = sim.NewGameState()
		g.paused = false
	}
	if g.paused || g.state.GameOver {
		return nil
	}

	input := sim.InputFrame{
		Team:   sim.TeamHome,
		Move:   movementVector(),
		Shoot:  inpututil.IsKeyJustPressed(ebiten.KeySpace),
		Pass:   inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight),
		Switch: inpututil.IsKeyJustPressed(ebiten.KeyTab),
	}
	sim.Step(&g.state, []sim.InputFrame{input})
	return nil
}

func movementVector() sim.Vec2 {
	move := sim.Vec2{}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		move.X -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		move.X += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		move.Y -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		move.Y += 1
	}
	return move.Normalized()
}

func (g *SoloGame) Draw(screen *ebiten.Image) {
	screen.Fill(colorHUDBackground)
	g.drawMatch(screen, sim.TeamHome)
	g.drawHUD(screen)
}

func (g *SoloGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return int(sim.WindowWidth), int(sim.WindowHeight)
}

func (g *SoloGame) drawMatch(screen *ebiten.Image, controlledTeam sim.Team) {
	homePalette := paletteForTeam(g.state, sim.TeamHome)
	awayPalette := paletteForTeam(g.state, sim.TeamAway)

	g.drawRink(screen)
	g.drawGoal(screen, true)
	g.drawGoal(screen, false)
	g.drawGoalie(screen, g.state.HomeGoalie, homePalette)
	g.drawGoalie(screen, g.state.AwayGoalie, awayPalette)

	for index, skater := range g.state.HomeSkaters {
		g.drawSkater(screen, skater, controlledTeam == sim.TeamHome && index == g.state.HomeControlled, homePalette)
	}
	for index, skater := range g.state.AwaySkaters {
		g.drawSkater(screen, skater, controlledTeam == sim.TeamAway && index == g.state.AwayControlled, awayPalette)
	}

	g.drawPuck(screen)
}

func (g *SoloGame) drawRink(screen *ebiten.Image) {
	drawRoundedFill(screen, sim.RinkLeft-16, sim.RinkTop-16, sim.RinkRight-sim.RinkLeft+32, sim.RinkBottom-sim.RinkTop+32, sim.RinkCornerRadius+16, colorBoard)
	drawRoundedFill(screen, sim.RinkLeft-10, sim.RinkTop-10, sim.RinkRight-sim.RinkLeft+20, sim.RinkBottom-sim.RinkTop+20, sim.RinkCornerRadius+10, colorBoardOutline)
	drawRoundedFill(screen, sim.RinkLeft, sim.RinkTop, sim.RinkRight-sim.RinkLeft, sim.RinkBottom-sim.RinkTop, sim.RinkCornerRadius, colorIce)

	drawLine(screen, sim.CenterX, sim.RinkTop, sim.CenterX, sim.RinkBottom, 4, colorCenterRed)
	drawLine(screen, sim.RinkLeft+240, sim.RinkTop, sim.RinkLeft+240, sim.RinkBottom, 5, colorBlueLine)
	drawLine(screen, sim.RinkRight-240, sim.RinkTop, sim.RinkRight-240, sim.RinkBottom, 5, colorBlueLine)
	drawLine(screen, sim.HomeGoalLineX, sim.RinkTop, sim.HomeGoalLineX, sim.RinkBottom, 2, colorCenterRed)
	drawLine(screen, sim.AwayGoalLineX, sim.RinkTop, sim.AwayGoalLineX, sim.RinkBottom, 2, colorCenterRed)

	vector.StrokeCircle(screen, float32(sim.CenterX), float32(sim.CenterY), 90, 3, colorCenterRed, true)
	vector.DrawFilledCircle(screen, float32(sim.CenterX), float32(sim.CenterY), 10, colorCenterRed, true)

	for _, circleX := range []float64{sim.RinkLeft + 180, sim.RinkRight - 180} {
		for _, circleY := range []float64{sim.CenterY - 140, sim.CenterY + 140} {
			vector.StrokeCircle(screen, float32(circleX), float32(circleY), 60, 2, colorCenterRed, true)
			vector.DrawFilledCircle(screen, float32(circleX), float32(circleY), 8, colorCenterRed, true)
		}
	}

	g.drawCrease(screen, true)
	g.drawCrease(screen, false)
}

func (g *SoloGame) drawCrease(screen *ebiten.Image, leftGoal bool) {
	goalX := sim.HomeGoalLineX
	if !leftGoal {
		goalX = sim.AwayGoalLineX
	}
	creaseRadius := 74.0
	vector.DrawFilledCircle(screen, float32(goalX), float32(sim.CenterY), float32(creaseRadius), colorCrease, true)
	vector.StrokeCircle(screen, float32(goalX), float32(sim.CenterY), float32(creaseRadius), 3, colorCreaseLine, true)
	if leftGoal {
		ebitenutil.DrawRect(screen, sim.RinkLeft-2, sim.CenterY-creaseRadius-4, goalX-sim.RinkLeft+3, creaseRadius*2+8, colorIce)
	} else {
		ebitenutil.DrawRect(screen, goalX, sim.CenterY-creaseRadius-4, sim.RinkRight-goalX+3, creaseRadius*2+8, colorIce)
	}
	drawLine(screen, goalX, sim.CenterY-creaseRadius, goalX, sim.CenterY+creaseRadius, 3, colorCreaseLine)
}

func (g *SoloGame) drawGoal(screen *ebiten.Image, leftGoal bool) {
	goalX := sim.HomeGoalLineX
	direction := -1.0
	if !leftGoal {
		goalX = sim.AwayGoalLineX
		direction = 1.0
	}
	goalTop := sim.CenterY - sim.GoalHalfHeight
	goalBottom := sim.CenterY + sim.GoalHalfHeight
	backX := goalX + sim.GoalDepth*direction
	backInset := 24.0
	backTop := goalTop + backInset
	backBottom := goalBottom - backInset

	for depth := 10.0; depth <= 30.0; depth += 10.0 {
		meshX := goalX + depth*direction
		blend := math.Abs(depth / sim.GoalDepth)
		meshTop := goalTop + (backTop-goalTop)*blend
		meshBottom := goalBottom + (backBottom-goalBottom)*blend
		drawLine(screen, meshX, meshTop+4, meshX, meshBottom-4, 1, colorNetMesh)
	}
	for meshY := backTop + 14.0; meshY < backBottom; meshY += 16.0 {
		blend := (meshY - backTop) / math.Max(backBottom-backTop, 1)
		frontY := goalTop + (goalBottom-goalTop)*blend
		drawLine(screen, goalX, frontY, backX, meshY, 1, colorNetMesh)
	}

	drawLine(screen, goalX, goalTop, backX, backTop, 3, colorNetFrame)
	drawLine(screen, goalX, goalBottom, backX, backBottom, 3, colorNetFrame)
	drawLine(screen, backX, backTop, backX, backBottom, 3, colorNetFrame)
	drawLine(screen, goalX, goalTop, goalX, goalBottom, 2, colorNetFrame)
}

func (g *SoloGame) drawSkater(screen *ebiten.Image, skater sim.SkaterState, controlled bool, palette teamPalette) {
	if controlled {
		vector.StrokeCircle(screen, float32(skater.Position.X), float32(skater.Position.Y), float32(skater.Radius+8), 4, palette.Primary, true)
	}
	vector.DrawFilledCircle(screen, float32(skater.Position.X), float32(skater.Position.Y), float32(skater.Radius), palette.Primary, true)
	vector.StrokeCircle(screen, float32(skater.Position.X), float32(skater.Position.Y), float32(skater.Radius-4), 3, palette.Trim, true)

	facing := skater.LookDir.Normalized()
	if facing.Length() < 0.2 {
		facing = sim.Vec2{X: 1}
		if skater.Team == sim.TeamAway {
			facing.X = -1
		}
	}
	stickStart := skater.Position.Add(facing.Mul(skater.Radius * 0.5))
	stickEnd := skater.Position.Add(facing.Mul(skater.Radius + 23))
	drawLine(screen, stickStart.X, stickStart.Y, stickEnd.X, stickEnd.Y, 5, color.RGBA{0x3f, 0x30, 0x20, 0xff})
	ebitenutil.DebugPrintAt(screen, string(skater.Role), int(skater.Position.X)-7, int(skater.Position.Y)-6)
}

func (g *SoloGame) drawGoalie(screen *ebiten.Image, goalie sim.GoalieState, palette teamPalette) {
	vector.DrawFilledCircle(screen, float32(goalie.Position.X), float32(goalie.Position.Y), float32(goalie.Radius), palette.Primary, true)
	ebitenutil.DrawRect(screen, goalie.Position.X-goalie.Radius+4, goalie.Position.Y+4, goalie.Radius*2-8, goalie.Radius-7, palette.Trim)
}

func (g *SoloGame) drawPuck(screen *ebiten.Image) {
	vector.DrawFilledCircle(screen, float32(g.state.Puck.Position.X), float32(g.state.Puck.Position.Y+3), float32(g.state.Puck.Radius), color.RGBA{0x6a, 0x71, 0x79, 0x80}, true)
	vector.DrawFilledCircle(screen, float32(g.state.Puck.Position.X), float32(g.state.Puck.Position.Y), float32(g.state.Puck.Radius), colorPuck, true)
}

func (g *SoloGame) drawHUD(screen *ebiten.Image) {
	minutes := g.state.ClockTicks / (sim.TickRate * 60)
	seconds := (g.state.ClockTicks / sim.TickRate) % 60
	periodLabel := fmt.Sprintf("P%d", g.state.Period)
	if g.state.InOvertime {
		periodLabel = "OT"
	}
	status := "Solo mode  WASD move  Shift pass  Space shoot/check  Tab switch  P pause  R restart"
	if g.paused {
		status = "Paused  Press P to resume or R to restart"
	}
	if g.state.GameOver {
		status = "Game over  Press R to restart solo play"
	}
	ebitenutil.DebugPrintAt(screen, "Hockey 26 v2 Solo", 20, 18)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d - %d", g.state.Score.Home, g.state.Score.Away), int(sim.CenterX)-24, 20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s %02d:%02d", periodLabel, minutes, seconds), 20, 42)
	ebitenutil.DebugPrintAt(screen, status, 20, int(sim.WindowHeight)-28)
}

func (g *SoloGame) drawReadyOverlay(screen *ebiten.Image, localTeam sim.Team, subtitle string) {
	ebitenutil.DrawRect(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, colorOverlay)
	panelWidth := 360.0
	gap := 34.0
	leftX := sim.CenterX - panelWidth - gap/2
	rightX := sim.CenterX + gap/2
	panelY := 170.0
	title := "Pregame"
	statusLine := "Both players must ready up to start"
	if g.state.Phase == sim.MatchPhaseIntermission {
		period := g.state.LastIntermissionStats.Period
		if period == 0 && g.state.Period > 1 {
			period = g.state.Period - 1
		}
		title = fmt.Sprintf("Intermission - End of Period %d", period)
		secondsLeft := (g.state.PhaseTicks + sim.TickRate - 1) / sim.TickRate
		statusLine = fmt.Sprintf("Auto resume in %ds", secondsLeft)
	}

	ebitenutil.DebugPrintAt(screen, title, int(sim.CenterX)-110, 54)
	ebitenutil.DebugPrintAt(screen, subtitle, int(sim.CenterX)-120, 80)
	ebitenutil.DebugPrintAt(screen, statusLine, int(sim.CenterX)-112, 104)
	ebitenutil.DebugPrintAt(screen, "A/Left and D/Right change color  Space or Enter toggles ready", int(sim.CenterX)-200, 128)

	g.drawTeamSelectionCard(screen, leftX, panelY, sim.TeamHome, localTeam == sim.TeamHome, g.state.HomeReady)
	g.drawTeamSelectionCard(screen, rightX, panelY, sim.TeamAway, localTeam == sim.TeamAway, g.state.AwayReady)
	if g.state.Phase == sim.MatchPhaseIntermission {
		g.drawIntermissionStatsCard(screen, sim.CenterX-230, 470)
	}
}

func (g *SoloGame) drawTeamSelectionCard(screen *ebiten.Image, x, y float64, team sim.Team, local bool, ready bool) {
	palette := paletteForTeam(g.state, team)
	cardWidth := 360.0
	cardHeight := 260.0
	ebitenutil.DrawRect(screen, x+6, y+8, cardWidth, cardHeight, colorPanelShadow)
	drawRoundedFill(screen, x, y, cardWidth, cardHeight, 22, colorPanel)
	ebitenutil.DrawRect(screen, x, y, cardWidth, 16, palette.Primary)
	vector.DrawFilledCircle(screen, float32(x+48), float32(y+62), 18, palette.Primary, true)
	vector.StrokeCircle(screen, float32(x+48), float32(y+62), 18, 3, palette.Trim, true)

	teamLabel := "HOME"
	if team == sim.TeamAway {
		teamLabel = "AWAY"
	}
	ownerLabel := "Opponent"
	if local {
		ownerLabel = "You"
	}
	readyLabel := "Not Ready"
	if ready {
		readyLabel = "Ready"
	}

	ebitenutil.DebugPrintAt(screen, teamLabel, int(x)+82, int(y)+36)
	ebitenutil.DebugPrintAt(screen, ownerLabel, int(x)+82, int(y)+58)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Color: %s", teamColorLabel(teamColorForDisplay(g.state, team))), int(x)+28, int(y)+106)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", readyLabel), int(x)+28, int(y)+132)
	if ready {
		ebitenutil.DebugPrintAt(screen, "Press Space/Enter again to unready", int(x)+28, int(y)+170)
	} else if local {
		ebitenutil.DebugPrintAt(screen, "Choose a color, then ready up", int(x)+28, int(y)+170)
	} else {
		ebitenutil.DebugPrintAt(screen, "Waiting on the other player", int(x)+28, int(y)+170)
	}
}

func (g *SoloGame) drawIntermissionStatsCard(screen *ebiten.Image, x, y float64) {
	stats := g.state.LastIntermissionStats
	if stats.Period == 0 {
		return
	}
	cardWidth := 460.0
	cardHeight := 126.0
	ebitenutil.DrawRect(screen, x+6, y+8, cardWidth, cardHeight, colorPanelShadow)
	drawRoundedFill(screen, x, y, cardWidth, cardHeight, 22, colorPanel)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Period %d Summary", stats.Period), int(x)+164, int(y)+18)
	ebitenutil.DebugPrintAt(screen, "Team            SOG   Goals", int(x)+38, int(y)+48)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HOME (%s)      %d      %d", teamColorLabel(g.state.HomeColor), stats.Home.ShotsOnGoal, stats.Home.Goals), int(x)+38, int(y)+72)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("AWAY (%s)      %d      %d", teamColorLabel(g.state.AwayColor), stats.Away.ShotsOnGoal, stats.Away.Goals), int(x)+38, int(y)+96)
}

func teamColorForDisplay(state sim.GameState, team sim.Team) sim.TeamColor {
	if team == sim.TeamHome {
		return state.HomeColor
	}
	return state.AwayColor
}

func drawRoundedFill(screen *ebiten.Image, x, y, width, height, radius float64, fill color.Color) {
	ebitenutil.DrawRect(screen, x+radius, y, width-radius*2, height, fill)
	ebitenutil.DrawRect(screen, x, y+radius, width, height-radius*2, fill)
	vector.DrawFilledCircle(screen, float32(x+radius), float32(y+radius), float32(radius), fill, true)
	vector.DrawFilledCircle(screen, float32(x+width-radius), float32(y+radius), float32(radius), fill, true)
	vector.DrawFilledCircle(screen, float32(x+radius), float32(y+height-radius), float32(radius), fill, true)
	vector.DrawFilledCircle(screen, float32(x+width-radius), float32(y+height-radius), float32(radius), fill, true)
}

func drawLine(screen *ebiten.Image, x1, y1, x2, y2, width float64, clr color.Color) {
	vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), float32(width), clr, true)
}
