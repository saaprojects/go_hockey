package render

import (
	"fmt"
	"image/color"
	"math"

	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
)

func DrawMatch(screen *ebiten.Image, state sim.GameState, controlledTeam sim.Team) {
	screen.Fill(colorHUDBackground)
	homePalette := paletteForTeam(state, sim.TeamHome)
	awayPalette := paletteForTeam(state, sim.TeamAway)

	drawRink(screen)
	drawGoal(screen, true)
	drawGoal(screen, false)
	drawGoalie(screen, state.HomeGoalie, homePalette)
	drawGoalie(screen, state.AwayGoalie, awayPalette)

	for index, skater := range state.HomeSkaters {
		drawSkater(screen, skater, controlledTeam == sim.TeamHome && index == state.HomeControlled, homePalette)
	}
	for index, skater := range state.AwaySkaters {
		drawSkater(screen, skater, controlledTeam == sim.TeamAway && index == state.AwayControlled, awayPalette)
	}

	drawPuck(screen, state.Puck)
}

func DrawSoloHUD(screen *ebiten.Image, state sim.GameState, status string) {
	drawHUD(screen, state, "Go Hockey Solo", status)
}

func DrawNetworkHUD(screen *ebiten.Image, state sim.GameState, scoreLabel, status string) {
	drawHUD(screen, state, scoreLabel, status)
}

func drawHUD(screen *ebiten.Image, state sim.GameState, title, status string) {
	minutes := state.ClockTicks / (sim.TickRate * 60)
	seconds := (state.ClockTicks / sim.TickRate) % 60
	periodLabel := fmt.Sprintf("P%d", state.Period)
	if state.InOvertime {
		periodLabel = "OT"
	}
	ebitenutil.DebugPrintAt(screen, title, 20, 18)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d - %d", state.Score.Home, state.Score.Away), int(sim.CenterX)-24, 20)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s %02d:%02d", periodLabel, minutes, seconds), 20, 42)
	ebitenutil.DebugPrintAt(screen, status, 20, int(sim.WindowHeight)-28)
}

func DrawReadyOverlay(screen *ebiten.Image, state sim.GameState, localTeam sim.Team, subtitle string) {
	ebitenutil.DrawRect(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, ui.OverlayColor)
	panelWidth := 360.0
	gap := 34.0
	leftX := sim.CenterX - panelWidth - gap/2
	rightX := sim.CenterX + gap/2
	panelY := 170.0
	title := "Pregame"
	statusLine := "Both players must ready up to start"
	if state.HomeColor == state.AwayColor {
		statusLine = "Choose different colors before play can continue"
	}
	if state.Phase == sim.MatchPhaseIntermission {
		period := state.LastIntermissionStats.Period
		if period == 0 && state.Period > 1 {
			period = state.Period - 1
		}
		title = fmt.Sprintf("Intermission - End of Period %d", period)
		secondsLeft := (state.PhaseTicks + sim.TickRate - 1) / sim.TickRate
		if state.HomeColor != state.AwayColor {
			statusLine = fmt.Sprintf("Auto resume in %ds", secondsLeft)
		}
	}

	ebitenutil.DebugPrintAt(screen, title, int(sim.CenterX)-110, 54)
	ebitenutil.DebugPrintAt(screen, subtitle, int(sim.CenterX)-120, 80)
	ebitenutil.DebugPrintAt(screen, statusLine, int(sim.CenterX)-152, 104)
	ebitenutil.DebugPrintAt(screen, "A/Left and D/Right or click arrows change color  Space/Enter or click Ready toggles ready", int(sim.CenterX)-260, 128)

	drawTeamSelectionCard(screen, state, leftX, panelY, sim.TeamHome, localTeam == sim.TeamHome, state.HomeReady)
	drawTeamSelectionCard(screen, state, rightX, panelY, sim.TeamAway, localTeam == sim.TeamAway, state.AwayReady)
	if state.Phase == sim.MatchPhaseIntermission {
		drawIntermissionStatsCard(screen, state, sim.CenterX-230, 470)
	}
}

func ReadyOverlayCardRect(team sim.Team) ui.Rect {
	panelWidth := 360.0
	gap := 34.0
	x := sim.CenterX - panelWidth - gap/2
	if team == sim.TeamAway {
		x = sim.CenterX + gap/2
	}
	return ui.Rect{X: x, Y: 170, W: panelWidth, H: 260}
}

func ReadyOverlayColorPrevRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 28, Y: card.Y + 196, W: 42, H: 36}
}

func ReadyOverlayColorLabelRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 82, Y: card.Y + 196, W: 196, H: 36}
}

func ReadyOverlayColorNextRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 290, Y: card.Y + 196, W: 42, H: 36}
}

func ReadyOverlayReadyRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 28, Y: card.Y + 236, W: card.W - 56, H: 32}
}

func drawTeamSelectionCard(screen *ebiten.Image, state sim.GameState, x, y float64, team sim.Team, local bool, ready bool) {
	palette := paletteForTeam(state, team)
	cardWidth := 360.0
	cardHeight := 260.0
	ebitenutil.DrawRect(screen, x+6, y+8, cardWidth, cardHeight, ui.PanelShadowColor)
	ui.DrawRoundedFill(screen, x, y, cardWidth, cardHeight, 22, ui.PanelColor)
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
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Color: %s", TeamColorLabel(teamColorForDisplay(state, team))), int(x)+106, int(y)+106)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", readyLabel), int(x)+28, int(y)+132)
	if ready {
		ebitenutil.DebugPrintAt(screen, "Press Space/Enter or click Ready to unready", int(x)+28, int(y)+170)
	} else if local {
		ebitenutil.DebugPrintAt(screen, "Choose a color, then click Ready", int(x)+28, int(y)+170)
	} else {
		ebitenutil.DebugPrintAt(screen, "Waiting on the other player", int(x)+28, int(y)+170)
	}
	if !local {
		return
	}

	cursorX, cursorY := ebiten.CursorPosition()
	prevRect := ReadyOverlayColorPrevRect(team)
	nextRect := ReadyOverlayColorNextRect(team)
	readyRect := ReadyOverlayReadyRect(team)
	colorLabelRect := ReadyOverlayColorLabelRect(team)
	ui.DrawOverlayButton(screen, prevRect, "<", ui.PointInRect(float64(cursorX), float64(cursorY), prevRect), false)
	ui.DrawRoundedFill(screen, colorLabelRect.X, colorLabelRect.Y, colorLabelRect.W, colorLabelRect.H, 14, color.RGBA{0xe5, 0xec, 0xf5, 0xff})
	ui.DrawText(screen, TeamColorLabel(teamColorForDisplay(state, team)), ui.SmallFace(), colorLabelRect.X+18, colorLabelRect.Y+10, ui.TextDarkColor)
	ui.DrawOverlayButton(screen, nextRect, ">", ui.PointInRect(float64(cursorX), float64(cursorY), nextRect), false)
	readyLabel = "Ready"
	if ready {
		readyLabel = "Unready"
	}
	ui.DrawOverlayButton(screen, readyRect, readyLabel, ui.PointInRect(float64(cursorX), float64(cursorY), readyRect), true)
}

func drawIntermissionStatsCard(screen *ebiten.Image, state sim.GameState, x, y float64) {
	stats := state.LastIntermissionStats
	if stats.Period == 0 {
		return
	}
	cardWidth := 460.0
	cardHeight := 126.0
	ebitenutil.DrawRect(screen, x+6, y+8, cardWidth, cardHeight, ui.PanelShadowColor)
	ui.DrawRoundedFill(screen, x, y, cardWidth, cardHeight, 22, ui.PanelColor)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Period %d Summary", stats.Period), int(x)+164, int(y)+18)
	ebitenutil.DebugPrintAt(screen, "Team            SOG   Goals", int(x)+38, int(y)+48)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HOME (%s)      %d      %d", TeamColorLabel(state.HomeColor), stats.Home.ShotsOnGoal, stats.Home.Goals), int(x)+38, int(y)+72)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("AWAY (%s)      %d      %d", TeamColorLabel(state.AwayColor), stats.Away.ShotsOnGoal, stats.Away.Goals), int(x)+38, int(y)+96)
}

func drawRink(screen *ebiten.Image) {
	ui.DrawRoundedFill(screen, sim.RinkLeft-16, sim.RinkTop-16, sim.RinkRight-sim.RinkLeft+32, sim.RinkBottom-sim.RinkTop+32, sim.RinkCornerRadius+16, colorBoard)
	ui.DrawRoundedFill(screen, sim.RinkLeft-10, sim.RinkTop-10, sim.RinkRight-sim.RinkLeft+20, sim.RinkBottom-sim.RinkTop+20, sim.RinkCornerRadius+10, colorBoardOutline)
	ui.DrawRoundedFill(screen, sim.RinkLeft, sim.RinkTop, sim.RinkRight-sim.RinkLeft, sim.RinkBottom-sim.RinkTop, sim.RinkCornerRadius, colorIce)

	ui.DrawLine(screen, sim.CenterX, sim.RinkTop, sim.CenterX, sim.RinkBottom, 4, colorCenterRed)
	ui.DrawLine(screen, sim.RinkLeft+240, sim.RinkTop, sim.RinkLeft+240, sim.RinkBottom, 5, colorBlueLine)
	ui.DrawLine(screen, sim.RinkRight-240, sim.RinkTop, sim.RinkRight-240, sim.RinkBottom, 5, colorBlueLine)
	ui.DrawLine(screen, sim.HomeGoalLineX, sim.RinkTop, sim.HomeGoalLineX, sim.RinkBottom, 2, colorCenterRed)
	ui.DrawLine(screen, sim.AwayGoalLineX, sim.RinkTop, sim.AwayGoalLineX, sim.RinkBottom, 2, colorCenterRed)

	vector.StrokeCircle(screen, float32(sim.CenterX), float32(sim.CenterY), 90, 3, colorCenterRed, true)
	vector.DrawFilledCircle(screen, float32(sim.CenterX), float32(sim.CenterY), 10, colorCenterRed, true)

	for _, circleX := range []float64{sim.RinkLeft + 180, sim.RinkRight - 180} {
		for _, circleY := range []float64{sim.CenterY - 140, sim.CenterY + 140} {
			vector.StrokeCircle(screen, float32(circleX), float32(circleY), 60, 2, colorCenterRed, true)
			vector.DrawFilledCircle(screen, float32(circleX), float32(circleY), 8, colorCenterRed, true)
		}
	}

	drawCrease(screen, true)
	drawCrease(screen, false)
}

func drawCrease(screen *ebiten.Image, leftGoal bool) {
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
	ui.DrawLine(screen, goalX, sim.CenterY-creaseRadius, goalX, sim.CenterY+creaseRadius, 3, colorCreaseLine)
}

func drawGoal(screen *ebiten.Image, leftGoal bool) {
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
		ui.DrawLine(screen, meshX, meshTop+4, meshX, meshBottom-4, 1, colorNetMesh)
	}
	for meshY := backTop + 14.0; meshY < backBottom; meshY += 16.0 {
		blend := (meshY - backTop) / math.Max(backBottom-backTop, 1)
		frontY := goalTop + (goalBottom-goalTop)*blend
		ui.DrawLine(screen, goalX, frontY, backX, meshY, 1, colorNetMesh)
	}

	ui.DrawLine(screen, goalX, goalTop, backX, backTop, 3, colorNetFrame)
	ui.DrawLine(screen, goalX, goalBottom, backX, backBottom, 3, colorNetFrame)
	ui.DrawLine(screen, backX, backTop, backX, backBottom, 3, colorNetFrame)
	ui.DrawLine(screen, goalX, goalTop, goalX, goalBottom, 2, colorNetFrame)
}

func drawSkater(screen *ebiten.Image, skater sim.SkaterState, controlled bool, palette teamPalette) {
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
	ui.DrawLine(screen, stickStart.X, stickStart.Y, stickEnd.X, stickEnd.Y, 5, color.RGBA{0x3f, 0x30, 0x20, 0xff})
	ebitenutil.DebugPrintAt(screen, string(skater.Role), int(skater.Position.X)-7, int(skater.Position.Y)-6)
}

func drawGoalie(screen *ebiten.Image, goalie sim.GoalieState, palette teamPalette) {
	vector.DrawFilledCircle(screen, float32(goalie.Position.X), float32(goalie.Position.Y), float32(goalie.Radius), palette.Primary, true)
	ebitenutil.DrawRect(screen, goalie.Position.X-goalie.Radius+4, goalie.Position.Y+4, goalie.Radius*2-8, goalie.Radius-7, palette.Trim)
}

func drawPuck(screen *ebiten.Image, puck sim.PuckState) {
	vector.DrawFilledCircle(screen, float32(puck.Position.X), float32(puck.Position.Y+3), float32(puck.Radius), color.RGBA{0x6a, 0x71, 0x79, 0x80}, true)
	vector.DrawFilledCircle(screen, float32(puck.Position.X), float32(puck.Position.Y), float32(puck.Radius), colorPuck, true)
}

func teamColorForDisplay(state sim.GameState, team sim.Team) sim.TeamColor {
	if team == sim.TeamHome {
		return state.HomeColor
	}
	return state.AwayColor
}
