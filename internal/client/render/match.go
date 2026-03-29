package render

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	colorHUDBackground = color.RGBA{0x07, 0x15, 0x28, 0xff}
	colorBoardOutline  = color.RGBA{0x9a, 0xae, 0xc1, 0xff}
	colorBoard         = color.RGBA{0xf9, 0xfb, 0xfd, 0xff}
	colorIce           = color.RGBA{0xde, 0xef, 0xfa, 0xff}
	colorCenterRed     = color.RGBA{0xd4, 0x4f, 0x56, 0xff}
	colorBlueLine      = color.RGBA{0x56, 0x98, 0xe5, 0xff}
	colorCrease        = color.RGBA{0xd0, 0xe7, 0xfb, 0xff}
	colorCreaseLine    = color.RGBA{0x6d, 0xa7, 0xe7, 0xff}
	colorNetFrame      = color.RGBA{0xdf, 0x46, 0x46, 0xff}
	colorNetMesh       = color.RGBA{0xcf, 0xd9, 0xe1, 0xff}
	colorPuck          = color.RGBA{0x10, 0x12, 0x15, 0xff}
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
	drawHUD(screen, state, "Solo Mode", status)
}

func DrawNetworkHUD(screen *ebiten.Image, state sim.GameState, scoreLabel, status string) {
	drawHUD(screen, state, scoreLabel, status)
}

func drawHUD(screen *ebiten.Image, state sim.GameState, title, status string) {
	minutes := state.ClockTicks / (sim.TickRate * 60)
	seconds := (state.ClockTicks / sim.TickRate) % 60
	periodLabel := fmt.Sprintf("PERIOD %d", state.Period)
	if state.InOvertime {
		periodLabel = "OVERTIME"
	}

	titleRect := ui.Rect{X: 18, Y: 14, W: 282, H: 40}
	scoreRect := ui.Rect{X: sim.CenterX - 112, Y: 10, W: 224, H: 48}
	clockRect := ui.Rect{X: sim.WindowWidth - 194, Y: 14, W: 176, H: 40}
	statusRect := ui.Rect{X: 18, Y: sim.WindowHeight - 44, W: sim.WindowWidth - 36, H: 28}

	ui.DrawRoundedPanel(screen, titleRect, 18, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)
	ui.DrawRoundedPanel(screen, scoreRect, 20, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelAltColor)
	ui.DrawRoundedPanel(screen, clockRect, 18, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)
	ui.DrawRoundedPanel(screen, statusRect, 14, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelInsetColor)

	ui.DrawText(screen, title, ui.BodyFace(), titleRect.X+16, titleRect.Y+10, ui.TextSoftColor)
	ui.DrawTextCentered(screen, fmt.Sprintf("%d  -  %d", state.Score.Home, state.Score.Away), ui.TitleFace(), sim.CenterX, scoreRect.Y+7, ui.TextLightColor)
	ui.DrawTextCentered(screen, fmt.Sprintf("%s  %02d:%02d", periodLabel, minutes, seconds), ui.SmallFace(), clockRect.X+clockRect.W/2, clockRect.Y+11, ui.TextSoftColor)
	ui.DrawTextCentered(screen, status, ui.SmallFace(), statusRect.X+statusRect.W/2, statusRect.Y+6, ui.TextSoftColor)
}

func DrawReadyOverlay(screen *ebiten.Image, state sim.GameState, localTeam sim.Team, subtitle string) {
	ui.DrawRoundedFill(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, 0, ui.OverlayColor)
	badge := ui.Rect{X: sim.CenterX - 274, Y: 34, W: 548, H: 70}
	ui.DrawGlow(screen, badge, 22, ui.WithAlpha(ui.AccentSoftColor, 58))
	ui.DrawRoundedPanel(screen, badge, 24, ui.PanelShadowColor, ui.PanelStrokeBrightColor, ui.PanelColor)

	title := "Pregame"
	statusLine := "Both players must ready up before the puck drops"
	if state.HomeColor == state.AwayColor {
		statusLine = "Teams need different colors before play can continue"
	}
	if state.Phase == sim.MatchPhaseIntermission {
		period := state.LastIntermissionStats.Period
		if period == 0 && state.Period > 1 {
			period = state.Period - 1
		}
		title = fmt.Sprintf("Intermission  |  End of Period %d", period)
		secondsLeft := (state.PhaseTicks + sim.TickRate - 1) / sim.TickRate
		if state.HomeColor != state.AwayColor {
			statusLine = fmt.Sprintf("Auto-resume in %ds unless someone changes colors", secondsLeft)
		}
	}

	ui.DrawTextCentered(screen, title, ui.TitleFace(), sim.CenterX, badge.Y+14, ui.TextLightColor)
	ui.DrawTextCentered(screen, subtitle, ui.BodyFace(), sim.CenterX, 118, ui.TextSoftColor)
	ui.DrawTextCentered(screen, statusLine, ui.SmallFace(), sim.CenterX, 144, ui.TextMutedColor)
	helpRect := ui.Rect{X: sim.CenterX - 360, Y: 156, W: 720, H: 30}
	ui.DrawRoundedPanel(screen, helpRect, 16, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 170), ui.PanelInsetColor)
	ui.DrawTextCentered(screen, "A/Left and D/Right change color. Space or Enter toggles ready. Mouse clicks work too.", ui.SmallFace(), sim.CenterX, helpRect.Y+7, ui.TextSoftColor)

	drawTeamSelectionCard(screen, state, sim.TeamHome, localTeam == sim.TeamHome, state.HomeReady)
	drawTeamSelectionCard(screen, state, sim.TeamAway, localTeam == sim.TeamAway, state.AwayReady)
	if state.Phase == sim.MatchPhaseIntermission {
		drawIntermissionStatsCard(screen, state, sim.CenterX-250, 486)
	}
}

func ReadyOverlayCardRect(team sim.Team) ui.Rect {
	panelWidth := 360.0
	gap := 34.0
	x := sim.CenterX - panelWidth - gap/2
	if team == sim.TeamAway {
		x = sim.CenterX + gap/2
	}
	return ui.Rect{X: x, Y: 198, W: panelWidth, H: 276}
}

func ReadyOverlayColorPrevRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 28, Y: card.Y + 206, W: 42, H: 36}
}

func ReadyOverlayColorLabelRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 82, Y: card.Y + 206, W: 196, H: 36}
}

func ReadyOverlayColorNextRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 290, Y: card.Y + 206, W: 42, H: 36}
}

func ReadyOverlayReadyRect(team sim.Team) ui.Rect {
	card := ReadyOverlayCardRect(team)
	return ui.Rect{X: card.X + 28, Y: card.Y + 248, W: card.W - 56, H: 40}
}

func drawTeamSelectionCard(screen *ebiten.Image, state sim.GameState, team sim.Team, local bool, ready bool) {
	area := ReadyOverlayCardRect(team)
	palette := paletteForTeam(state, team)
	if local {
		ui.DrawGlow(screen, area, 22, ui.WithAlpha(ui.AccentSoftColor, 54))
	}
	ui.DrawRoundedPanel(screen, area, 24, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)
	vector.FillRect(screen, float32(area.X+18), float32(area.Y+16), float32(area.W-36), 1, ui.FrostLineColor, false)
	vector.FillRect(screen, float32(area.X), float32(area.Y), float32(area.W), 12, palette.Primary, false)

	teamLabel := "HOME TEAM"
	ownerLabel := "Opponent"
	if team == sim.TeamAway {
		teamLabel = "AWAY TEAM"
	}
	if local {
		ownerLabel = "You"
	}
	readyLabel := "Not Ready"
	statusFill := ui.PanelInsetColor
	statusOutline := ui.PanelStrokeColor
	statusText := ui.TextSoftColor
	if ready {
		readyLabel = "Ready"
		statusFill = ui.AccentDeepColor
		statusOutline = ui.AccentColor
		statusText = ui.TextLightColor
	}

	vector.FillCircle(screen, float32(area.X+38), float32(area.Y+48), 12, palette.Primary, true)
	vector.StrokeCircle(screen, float32(area.X+38), float32(area.Y+48), 12, 2, palette.Trim, true)
	ui.DrawText(screen, teamLabel, ui.BodyFace(), area.X+58, area.Y+22, ui.TextLightColor)
	ui.DrawText(screen, ownerLabel, ui.SmallFace(), area.X+58, area.Y+52, ui.TextMutedColor)

	statusRect := ui.Rect{X: area.X + area.W - 132, Y: area.Y + 26, W: 104, H: 28}
	ui.DrawRoundedPanel(screen, statusRect, 14, color.RGBA{0, 0, 0, 0}, statusOutline, statusFill)
	ui.DrawTextCentered(screen, strings.ToUpper(readyLabel), ui.SmallFace(), statusRect.X+statusRect.W/2, statusRect.Y+7, statusText)

	ui.DrawText(screen, "Team Color", ui.SmallFace(), area.X+28, area.Y+92, ui.TextMutedColor)
	colorLabelArea := ui.Rect{X: area.X + 28, Y: area.Y + 114, W: 156, H: 40}
	ui.DrawRoundedPanel(screen, colorLabelArea, 18, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 190), ui.PanelAltColor)
	vector.FillCircle(screen, float32(colorLabelArea.X+20), float32(colorLabelArea.Y+20), 10, palette.Primary, true)
	vector.StrokeCircle(screen, float32(colorLabelArea.X+20), float32(colorLabelArea.Y+20), 10, 2, palette.Trim, true)
	ui.DrawText(screen, TeamColorLabel(teamColorForDisplay(state, team)), ui.BodyFace(), colorLabelArea.X+40, colorLabelArea.Y+9, ui.TextLightColor)

	helper := "Waiting for the other player."
	if local && ready {
		helper = "Ready selected. You can still unready or swap colors."
	} else if local {
		helper = "Choose a color, then hit Ready when you are set."
	}
	ui.DrawText(screen, helper, ui.SmallFace(), area.X+28, area.Y+168, ui.TextSoftColor)
	ui.DrawText(screen, "Intermissions use the same controls so rematches stay quick.", ui.TinyFace(), area.X+28, area.Y+188, ui.TextMutedColor)
	if !local {
		return
	}

	cursorX, cursorY := ebiten.CursorPosition()
	prevRect := ReadyOverlayColorPrevRect(team)
	nextRect := ReadyOverlayColorNextRect(team)
	readyRect := ReadyOverlayReadyRect(team)
	textRect := ReadyOverlayColorLabelRect(team)
	ui.DrawOverlayButton(screen, prevRect, "<", ui.PointInRect(float64(cursorX), float64(cursorY), prevRect), false)
	ui.DrawRoundedPanel(screen, textRect, 16, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 180), ui.PanelInsetColor)
	ui.DrawTextCentered(screen, TeamColorLabel(teamColorForDisplay(state, team)), ui.BodyFace(), textRect.X+textRect.W/2, textRect.Y+8, ui.TextLightColor)
	ui.DrawOverlayButton(screen, nextRect, ">", ui.PointInRect(float64(cursorX), float64(cursorY), nextRect), false)
	buttonLabel := "READY"
	if ready {
		buttonLabel = "UNREADY"
	}
	ui.DrawOverlayButton(screen, readyRect, buttonLabel, ui.PointInRect(float64(cursorX), float64(cursorY), readyRect), true)
}

func drawIntermissionStatsCard(screen *ebiten.Image, state sim.GameState, x, y float64) {
	stats := state.LastIntermissionStats
	if stats.Period == 0 {
		return
	}
	area := ui.Rect{X: x, Y: y, W: 500, H: 138}
	ui.DrawGlow(screen, area, 22, ui.WithAlpha(ui.AccentSoftColor, 40))
	ui.DrawRoundedPanel(screen, area, 24, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)
	ui.DrawTextCentered(screen, fmt.Sprintf("Period %d Summary", stats.Period), ui.HeadingFace(), area.X+area.W/2, area.Y+16, ui.TextLightColor)
	ui.DrawText(screen, "Team", ui.SmallFace(), area.X+34, area.Y+58, ui.TextMutedColor)
	ui.DrawText(screen, "Shots", ui.SmallFace(), area.X+252, area.Y+58, ui.TextMutedColor)
	ui.DrawText(screen, "Goals", ui.SmallFace(), area.X+350, area.Y+58, ui.TextMutedColor)
	ui.DrawText(screen, fmt.Sprintf("Home (%s)", TeamColorLabel(state.HomeColor)), ui.BodyFace(), area.X+34, area.Y+80, ui.TextLightColor)
	ui.DrawText(screen, fmt.Sprintf("%d", stats.Home.ShotsOnGoal), ui.BodyFace(), area.X+268, area.Y+80, ui.TextLightColor)
	ui.DrawText(screen, fmt.Sprintf("%d", stats.Home.Goals), ui.BodyFace(), area.X+366, area.Y+80, ui.TextLightColor)
	ui.DrawText(screen, fmt.Sprintf("Away (%s)", TeamColorLabel(state.AwayColor)), ui.BodyFace(), area.X+34, area.Y+108, ui.TextSoftColor)
	ui.DrawText(screen, fmt.Sprintf("%d", stats.Away.ShotsOnGoal), ui.BodyFace(), area.X+268, area.Y+108, ui.TextSoftColor)
	ui.DrawText(screen, fmt.Sprintf("%d", stats.Away.Goals), ui.BodyFace(), area.X+366, area.Y+108, ui.TextSoftColor)
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
	vector.FillCircle(screen, float32(sim.CenterX), float32(sim.CenterY), 10, colorCenterRed, true)

	for _, circleX := range []float64{sim.RinkLeft + 180, sim.RinkRight - 180} {
		for _, circleY := range []float64{sim.CenterY - 140, sim.CenterY + 140} {
			vector.StrokeCircle(screen, float32(circleX), float32(circleY), 60, 2, colorCenterRed, true)
			vector.FillCircle(screen, float32(circleX), float32(circleY), 8, colorCenterRed, true)
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
	vector.FillCircle(screen, float32(goalX), float32(sim.CenterY), float32(creaseRadius), colorCrease, true)
	vector.StrokeCircle(screen, float32(goalX), float32(sim.CenterY), float32(creaseRadius), 3, colorCreaseLine, true)
	if leftGoal {
		vector.FillRect(screen, float32(sim.RinkLeft-2), float32(sim.CenterY-creaseRadius-4), float32(goalX-sim.RinkLeft+3), float32(creaseRadius*2+8), colorIce, false)
	} else {
		vector.FillRect(screen, float32(goalX), float32(sim.CenterY-creaseRadius-4), float32(sim.RinkRight-goalX+3), float32(creaseRadius*2+8), colorIce, false)
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
		vector.StrokeCircle(screen, float32(skater.Position.X), float32(skater.Position.Y), float32(skater.Radius+8), 4, palette.Trim, true)
	}
	vector.FillCircle(screen, float32(skater.Position.X), float32(skater.Position.Y), float32(skater.Radius), palette.Primary, true)
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
}

func drawGoalie(screen *ebiten.Image, goalie sim.GoalieState, palette teamPalette) {
	vector.FillCircle(screen, float32(goalie.Position.X), float32(goalie.Position.Y), float32(goalie.Radius), palette.Primary, true)
	vector.StrokeCircle(screen, float32(goalie.Position.X), float32(goalie.Position.Y), float32(goalie.Radius-3), 3, palette.Trim, true)
	vector.FillRect(screen, float32(goalie.Position.X-goalie.Radius+4), float32(goalie.Position.Y+4), float32(goalie.Radius*2-8), float32(goalie.Radius-7), palette.Trim, false)
}

func drawPuck(screen *ebiten.Image, puck sim.PuckState) {
	vector.FillCircle(screen, float32(puck.Position.X), float32(puck.Position.Y+3), float32(puck.Radius), color.RGBA{0x6a, 0x71, 0x79, 0x80}, true)
	vector.FillCircle(screen, float32(puck.Position.X), float32(puck.Position.Y), float32(puck.Radius), colorPuck, true)
}

func teamColorForDisplay(state sim.GameState, team sim.Team) sim.TeamColor {
	if team == sim.TeamHome {
		return state.HomeColor
	}
	return state.AwayColor
}
