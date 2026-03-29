package sim

import (
	"strings"
	"testing"
)

func TestNewGameStateStartsWithRostersAndFaceoff(t *testing.T) {
	state := NewGameState()
	if len(state.HomeSkaters) != 3 {
		t.Fatalf("expected 3 home skaters, got %d", len(state.HomeSkaters))
	}
	if len(state.AwaySkaters) != 3 {
		t.Fatalf("expected 3 away skaters, got %d", len(state.AwaySkaters))
	}
	if state.Period != 1 {
		t.Fatalf("expected period 1, got %d", state.Period)
	}
	if state.FaceoffTicks <= 0 {
		t.Fatalf("expected active faceoff freeze, got %d", state.FaceoffTicks)
	}
	if state.Puck.Position.X != CenterX || state.Puck.Position.Y != CenterY {
		t.Fatalf("expected puck at center, got %#v", state.Puck.Position)
	}
}

func TestStepReleasesPuckAfterFaceoffCountdown(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 1
	Step(&state, nil)
	if state.Tick != 1 {
		t.Fatalf("expected tick 1, got %d", state.Tick)
	}
	if state.FaceoffTicks != 0 {
		t.Fatalf("expected faceoff to finish, got %d", state.FaceoffTicks)
	}
	if state.Puck.Velocity.X == 0 {
		t.Fatalf("expected puck to be released after faceoff")
	}
}

func TestSwitchControlChoosesClosestSkaterToPuck(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.HomeControlled = 0
	state.HomeSkaters[0].Position = Vec2{X: 250, Y: 250}
	state.HomeSkaters[1].Position = Vec2{X: 500, Y: 500}
	state.HomeSkaters[2].Position = Vec2{X: 645, Y: 405}
	state.Puck.Position = Vec2{X: 640, Y: 405}

	Step(&state, []InputFrame{{Team: TeamHome, Switch: true}})

	if state.HomeControlled != 2 {
		t.Fatalf("expected controlled skater 2, got %d", state.HomeControlled)
	}
}

func TestPassReleasesPuckFromCarrier(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	carrier := &state.HomeSkaters[state.HomeControlled]
	carrier.ActionCooldownTicks = 0
	state.Puck.CarrierID = carrier.ID
	state.Puck.Position = carrier.Position

	Step(&state, []InputFrame{{Team: TeamHome, Pass: true}})

	if state.Puck.CarrierID != "" {
		t.Fatalf("expected puck to be released, still carried by %q", state.Puck.CarrierID)
	}
	if state.Puck.Velocity.Length() == 0 {
		t.Fatalf("expected non-zero pass velocity")
	}
	if state.Puck.LastTouchTeam != TeamHome {
		t.Fatalf("expected home to be last touch, got %q", state.Puck.LastTouchTeam)
	}
}

func TestAwayCarrierUsesAIWithoutAwayInput(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	carrier := &state.AwaySkaters[state.AwayControlled]
	carrier.ActionCooldownTicks = 0
	state.Puck.CarrierID = carrier.ID
	state.Puck.Position = carrier.Position

	Step(&state, nil)

	if carrier.Velocity.Length() == 0 {
		t.Fatalf("expected away carrier to keep skating under AI control")
	}
}

func TestCrossingGoalLineScores(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.Puck.Position = Vec2{X: AwayGoalLineX - 6, Y: CenterY}
	state.Puck.Velocity = Vec2{X: 900, Y: 0}

	Step(&state, nil)

	if state.Score.Home != 1 {
		t.Fatalf("expected home score 1, got %+v", state.Score)
	}
	if state.GoalPauseTicks <= 0 {
		t.Fatalf("expected a short goal pause before the faceoff, got %d", state.GoalPauseTicks)
	}
	pauseTicks := state.GoalPauseTicks
	for step := 0; step < pauseTicks; step++ {
		Step(&state, nil)
	}
	if state.FaceoffTicks <= 0 {
		t.Fatalf("expected faceoff reset after the goal pause")
	}
}
func TestGoalPauseUsesScoringTeamColorDuration(t *testing.T) {
	tests := []struct {
		name      string
		homeColor TeamColor
		awayColor TeamColor
		position  Vec2
		velocity  Vec2
		want      int
	}{
		{
			name:      "home orange goal uses orange horn length",
			homeColor: TeamColorOrange,
			awayColor: TeamColorBlue,
			position:  Vec2{X: AwayGoalLineX - 6, Y: CenterY},
			velocity:  Vec2{X: 900, Y: 0},
			want:      ticksFromSeconds(GoalPauseOrangeSeconds),
		},
		{
			name:      "away red goal uses red horn length",
			homeColor: TeamColorGreen,
			awayColor: TeamColorRed,
			position:  Vec2{X: HomeGoalLineX + 6, Y: CenterY},
			velocity:  Vec2{X: -900, Y: 0},
			want:      ticksFromSeconds(GoalPauseRedSeconds),
		},
	}

	for _, tc := range tests {
		state := NewGameState()
		state.FaceoffTicks = 0
		state.HomeColor = tc.homeColor
		state.AwayColor = tc.awayColor
		state.Puck.Position = tc.position
		state.Puck.Velocity = tc.velocity

		Step(&state, nil)

		if state.GoalPauseTicks != tc.want {
			t.Fatalf("%s: expected goal pause %d, got %d", tc.name, tc.want, state.GoalPauseTicks)
		}
	}
}

func TestDefensiveAssignmentsCapPressureAtTwo(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	enemyCarrier := &state.AwaySkaters[1]
	enemyCarrier.Position = Vec2{X: CenterX - 110, Y: CenterY - 8}
	state.AwaySkaters[0].Position = Vec2{X: CenterX - 245, Y: CenterY - 120}
	state.AwaySkaters[2].Position = Vec2{X: CenterX - 230, Y: CenterY + 135}
	state.Puck.CarrierID = enemyCarrier.ID
	state.Puck.Position = enemyCarrier.Position

	state.HomeSkaters[0].Position = Vec2{X: CenterX - 165, Y: CenterY - 95}
	state.HomeSkaters[1].Position = Vec2{X: CenterX - 80, Y: CenterY + 18}
	state.HomeSkaters[2].Position = Vec2{X: CenterX - 285, Y: CenterY + 140}

	pressureCount := 0
	for index := range state.HomeSkaters {
		target := aiTarget(&state, &state.HomeSkaters[index])
		if target.Sub(enemyCarrier.Position).Length() <= 100.0 {
			pressureCount++
		}
	}

	if pressureCount != 2 {
		t.Fatalf("expected exactly 2 defenders pressuring, got %d", pressureCount)
	}
}

func TestAIPassPrefersOpenReceiver(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	carrier := &state.HomeSkaters[1]
	carrier.Position = Vec2{X: CenterX - 60, Y: CenterY}
	carrier.LookDir = Vec2{X: 1, Y: 0}
	state.Puck.CarrierID = carrier.ID
	state.Puck.Position = carrier.Position

	blocked := &state.HomeSkaters[0]
	blocked.Position = Vec2{X: CenterX + 120, Y: CenterY - 120}
	open := &state.HomeSkaters[2]
	open.Position = Vec2{X: CenterX + 70, Y: CenterY + 155}

	state.AwaySkaters[0].Position = Vec2{X: CenterX + 35, Y: CenterY - 25}
	state.AwaySkaters[1].Position = Vec2{X: CenterX + 85, Y: CenterY - 95}
	state.AwaySkaters[2].Position = Vec2{X: CenterX - 20, Y: CenterY + 10}

	passPuck(&state, carrier, Vec2{}, false)
	if state.Puck.CarrierID != "" {
		t.Fatalf("expected AI pass to release the puck")
	}

	passDir := state.Puck.Velocity.Normalized()
	openDir := open.Position.Sub(carrier.Position).Normalized()
	blockedDir := blocked.Position.Sub(carrier.Position).Normalized()
	if passDir.Dot(openDir) <= passDir.Dot(blockedDir) {
		t.Fatalf("expected AI pass to favor the open receiver")
	}
}

func TestPuckEnteringNetFromBehindDoesNotScore(t *testing.T) {
	tests := []struct {
		name     string
		previous Vec2
		current  Vec2
	}{
		{
			name:     "right goal",
			previous: Vec2{X: AwayGoalLineX + GoalDepth + 6, Y: CenterY},
			current:  Vec2{X: AwayGoalLineX + GoalDepth - 6, Y: CenterY},
		},
		{
			name:     "left goal",
			previous: Vec2{X: HomeGoalLineX - GoalDepth - 6, Y: CenterY},
			current:  Vec2{X: HomeGoalLineX - GoalDepth + 6, Y: CenterY},
		},
		{
			name:     "right goal vertical from behind",
			previous: Vec2{X: AwayGoalLineX + GoalDepth + 6, Y: CenterY - GoalHalfHeight - 14},
			current:  Vec2{X: AwayGoalLineX + GoalDepth + 6, Y: CenterY},
		},
		{
			name:     "left goal vertical from behind",
			previous: Vec2{X: HomeGoalLineX - GoalDepth - 6, Y: CenterY - GoalHalfHeight - 14},
			current:  Vec2{X: HomeGoalLineX - GoalDepth - 6, Y: CenterY},
		},
	}

	for _, tc := range tests {
		if scoringTeam, scored := checkGoalScored(tc.previous, tc.current); scored {
			t.Fatalf("%s: expected no goal from behind the net, got %q", tc.name, scoringTeam)
		}
	}
}

func TestPuckEnteringGoalMouthFromFrontScores(t *testing.T) {
	tests := []struct {
		name     string
		previous Vec2
		current  Vec2
		team     Team
	}{
		{
			name:     "right goal vertical entry",
			previous: Vec2{X: AwayGoalLineX, Y: CenterY - GoalHalfHeight - 14},
			current:  Vec2{X: AwayGoalLineX, Y: CenterY},
			team:     TeamHome,
		},
		{
			name:     "left goal vertical entry",
			previous: Vec2{X: HomeGoalLineX, Y: CenterY - GoalHalfHeight - 14},
			current:  Vec2{X: HomeGoalLineX, Y: CenterY},
			team:     TeamAway,
		},
		{
			name:     "right goal near top post",
			previous: Vec2{X: AwayGoalLineX - 18, Y: CenterY - GoalHalfHeight - 16},
			current:  Vec2{X: AwayGoalLineX - 3, Y: CenterY - GoalHalfHeight - 4},
			team:     TeamHome,
		},
		{
			name:     "left goal near bottom post",
			previous: Vec2{X: HomeGoalLineX + 18, Y: CenterY + GoalHalfHeight + 16},
			current:  Vec2{X: HomeGoalLineX + 3, Y: CenterY + GoalHalfHeight + 4},
			team:     TeamAway,
		},
		{
			name:     "right goal front edge catches lower mouth",
			previous: Vec2{X: AwayGoalLineX - 18, Y: CenterY + GoalHalfHeight + 18},
			current:  Vec2{X: AwayGoalLineX - 3, Y: CenterY + GoalHalfHeight + 14},
			team:     TeamHome,
		},
	}

	for _, tc := range tests {
		scoringTeam, scored := checkGoalScored(tc.previous, tc.current)
		if !scored || scoringTeam != tc.team {
			t.Fatalf("%s: expected %q to score, got scored=%v team=%q", tc.name, tc.team, scored, scoringTeam)
		}
	}
}

func TestPuckOutsidePostsStillDoesNotScore(t *testing.T) {
	tests := []struct {
		name     string
		previous Vec2
		current  Vec2
	}{
		{
			name:     "right goal above post",
			previous: Vec2{X: AwayGoalLineX - 18, Y: CenterY - GoalHalfHeight - 24},
			current:  Vec2{X: AwayGoalLineX - 3, Y: CenterY - GoalHalfHeight - 18},
		},
		{
			name:     "left goal below post",
			previous: Vec2{X: HomeGoalLineX + 18, Y: CenterY + GoalHalfHeight + 24},
			current:  Vec2{X: HomeGoalLineX + 3, Y: CenterY + GoalHalfHeight + 18},
		},
	}

	for _, tc := range tests {
		if scoringTeam, scored := checkGoalScored(tc.previous, tc.current); scored {
			t.Fatalf("%s: expected no goal outside the posts, got %q", tc.name, scoringTeam)
		}
	}
}

func TestCarriedPuckBehindNetDoesNotScore(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.HomeControlled = 1
	carrier := &state.HomeSkaters[state.HomeControlled]
	carrier.Position = Vec2{X: AwayGoalLineX + GoalDepth + 14, Y: CenterY}
	carrier.LookDir = Vec2{X: -1, Y: 0}
	state.Puck.CarrierID = carrier.ID
	state.Puck.Position = Vec2{X: AwayGoalLineX + GoalDepth + 10, Y: CenterY}

	Step(&state, []InputFrame{{Team: TeamHome}})

	if state.Score.Home != 0 {
		t.Fatalf("expected no home goal from behind the net, got %+v", state.Score)
	}

	state = NewGameState()
	state.FaceoffTicks = 0
	state.AwayControlled = 1
	carrier = &state.AwaySkaters[state.AwayControlled]
	carrier.Position = Vec2{X: HomeGoalLineX - GoalDepth - 14, Y: CenterY}
	carrier.LookDir = Vec2{X: 1, Y: 0}
	state.Puck.CarrierID = carrier.ID
	state.Puck.Position = Vec2{X: HomeGoalLineX - GoalDepth - 10, Y: CenterY}

	Step(&state, []InputFrame{{Team: TeamAway}})

	if state.Score.Away != 0 {
		t.Fatalf("expected no away goal from behind the net, got %+v", state.Score)
	}
}
func TestLoosePuckPickedUpAcrossGoalLineScores(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.HomeControlled = 1
	skater := &state.HomeSkaters[state.HomeControlled]
	skater.Position = Vec2{X: AwayGoalLineX - (skater.Radius + state.Puck.Radius + 2.0), Y: CenterY - 70.0}
	skater.LookDir = Vec2{X: 1, Y: 0}
	skater.Velocity = Vec2{}
	state.Puck.CarrierID = ""
	state.Puck.Position = skater.Position
	state.Puck.Velocity = Vec2{}
	state.Puck.PickupLockTeam = TeamNone
	state.Puck.PickupLockTicks = 0

	Step(&state, []InputFrame{{Team: TeamHome}})

	if state.Score.Home != 1 {
		t.Fatalf("expected loose puck pickup at the goal mouth to score, got %+v", state.Score)
	}
	if state.GoalPauseTicks <= 0 {
		t.Fatalf("expected a short goal pause after pickup scoring, got %d", state.GoalPauseTicks)
	}
}

func TestCarriedPuckBehindNetStaysOutOfGoalPocket(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.HomeControlled = 1
	carrier := &state.HomeSkaters[state.HomeControlled]
	carrier.Position = Vec2{X: AwayGoalLineX + GoalDepth + 18, Y: CenterY}
	carrier.LookDir = Vec2{X: -1, Y: 0}
	state.Puck.CarrierID = carrier.ID
	state.Puck.Position = carrier.Position

	Step(&state, []InputFrame{{Team: TeamHome}})

	if pointInsideGoal(state.Puck.Position, false) {
		t.Fatalf("expected carried puck to be pushed out of the right goal pocket, got %#v", state.Puck.Position)
	}
	if state.Score.Home != 0 {
		t.Fatalf("expected no goal from a behind-net carry, got %+v", state.Score)
	}
}

func TestSkaterInsideGoalPocketGetsPushedOut(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.HomeControlled = 1
	skater := &state.HomeSkaters[state.HomeControlled]
	skater.Position = Vec2{X: AwayGoalLineX + GoalDepth*0.55, Y: CenterY}
	skater.Velocity = Vec2{X: -180, Y: 0}
	state.Puck.CarrierID = ""
	state.Puck.Position = Vec2{X: CenterX, Y: CenterY}
	state.Puck.Velocity = Vec2{}
	for index := range state.AwaySkaters {
		state.AwaySkaters[index].Position = Vec2{X: CenterX + 180, Y: CenterY - 120 + float64(index)*120.0}
	}

	Step(&state, []InputFrame{{Team: TeamHome}})

	if pointInsideGoal(skater.Position, false) {
		t.Fatalf("expected skater to be pushed out of the right goal pocket, got %#v", skater.Position)
	}
}

func TestLoosePuckTrappedBehindNetForcesFaceoffAfterThreeSeconds(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.Puck.Position = Vec2{X: AwayGoalLineX + GoalDepth + state.Puck.Radius + 2.0, Y: CenterY}
	state.Puck.Velocity = Vec2{}
	state.Puck.PickupLockTeam = TeamHome
	state.Puck.PickupLockTicks = ticksFromSeconds(GoalTrapFaceoff) + 5
	for index := range state.HomeSkaters {
		state.HomeSkaters[index].Position = Vec2{X: CenterX - 220, Y: CenterY - 120 + float64(index)*120.0}
	}
	for index := range state.AwaySkaters {
		state.AwaySkaters[index].Position = Vec2{X: CenterX + 120, Y: CenterY - 120 + float64(index)*120.0}
	}

	for step := 0; step <= ticksFromSeconds(GoalTrapFaceoff); step++ {
		Step(&state, nil)
	}

	if state.FaceoffTicks <= 0 {
		t.Fatalf("expected a stuck puck behind the net to trigger a faceoff, got %+v", state)
	}
	if state.Puck.Position != (Vec2{X: CenterX, Y: CenterY}) {
		t.Fatalf("expected stuck-puck faceoff to reset the puck to center, got %#v", state.Puck.Position)
	}
}

func TestClockAdvancesToNextRegulationPeriod(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.ClockTicks = 1
	state.Score.Home = 1

	Step(&state, nil)

	if state.GameOver {
		t.Fatalf("expected game to continue into the next period")
	}
	if state.InOvertime {
		t.Fatalf("did not expect overtime in regulation")
	}
	if state.Period != 2 {
		t.Fatalf("expected period 2, got %d", state.Period)
	}
	if state.ClockTicks != ticksFromSeconds(PeriodLengthSeconds) {
		t.Fatalf("expected clock reset for next period, got %d", state.ClockTicks)
	}
	if state.FaceoffTicks == 0 {
		t.Fatalf("expected faceoff reset at intermission")
	}
}

func TestClockEntersOvertimeAfterTiedRegulation(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.Period = RegulationPeriods
	state.ClockTicks = 1
	state.Score.Home = 2
	state.Score.Away = 2

	Step(&state, nil)

	if !state.InOvertime {
		t.Fatalf("expected overtime after tied regulation")
	}
	if state.GameOver {
		t.Fatalf("expected tied game to continue into overtime")
	}
	if state.Period != RegulationPeriods+1 {
		t.Fatalf("expected overtime period marker %d, got %d", RegulationPeriods+1, state.Period)
	}
	if state.ClockTicks != ticksFromSeconds(OTLengthSeconds) {
		t.Fatalf("expected overtime clock reset, got %d", state.ClockTicks)
	}
}

func TestNewMultiplayerGameStateStartsInPregame(t *testing.T) {
	state := NewMultiplayerGameState()
	if !state.UseMenus {
		t.Fatalf("expected multiplayer state to enable menus")
	}
	if state.Phase != MatchPhasePregame {
		t.Fatalf("expected pregame phase, got %q", state.Phase)
	}
	if state.HomeColor != TeamColorBlue || state.AwayColor != TeamColorRed {
		t.Fatalf("expected default blue/red colors, got %q and %q", state.HomeColor, state.AwayColor)
	}
	if state.PhaseTicks != 0 {
		t.Fatalf("expected pregame to wait without a countdown, got %d", state.PhaseTicks)
	}
}

func TestPregameWaitsForBothPlayers(t *testing.T) {
	state := NewMultiplayerGameState()
	for index := 0; index < TickRate*2; index++ {
		Step(&state, nil)
	}
	if state.Phase != MatchPhasePregame {
		t.Fatalf("expected pregame to keep waiting, got %q", state.Phase)
	}
}

func TestPregameColorSelectionUsesStableCycle(t *testing.T) {
	state := NewMultiplayerGameState()
	state.HomeColor = TeamColorBlack
	state.AwayColor = TeamColorOrange

	Step(&state, []InputFrame{{Team: TeamHome, ColorNext: true}})

	if state.HomeColor != TeamColorOrange {
		t.Fatalf("expected home color to advance to orange, got %q", state.HomeColor)
	}
}

func TestPregameSecondReadyIsBlockedWhenColorsMatch(t *testing.T) {
	state := NewMultiplayerGameState()
	state.HomeColor = TeamColorBlue
	state.AwayColor = TeamColorBlue

	Step(&state, []InputFrame{{Team: TeamHome, Ready: true}})
	if !state.HomeReady {
		t.Fatalf("expected first team to be able to lock its current color")
	}

	Step(&state, []InputFrame{{Team: TeamAway, Ready: true}})
	if state.AwayReady {
		t.Fatalf("expected second team to stay unready when colors match")
	}
	if state.Phase != MatchPhasePregame {
		t.Fatalf("expected pregame to continue until colors differ, got %q", state.Phase)
	}
}

func TestPregameReadyStartsPlayImmediately(t *testing.T) {
	state := NewMultiplayerGameState()
	Step(&state, []InputFrame{{Team: TeamHome, Ready: true}, {Team: TeamAway, Ready: true}})
	if state.Phase != MatchPhasePlaying {
		t.Fatalf("expected match to start when both players are ready, got %q", state.Phase)
	}
}

func TestClockStartsIntermissionMenuWhenEnabled(t *testing.T) {
	state := NewMultiplayerGameState()
	state.Phase = MatchPhasePlaying
	state.PhaseTicks = 0
	state.FaceoffTicks = 0
	state.ClockTicks = 1
	state.Score.Home = 1

	Step(&state, nil)

	if state.GameOver {
		t.Fatalf("expected intermission instead of game over")
	}
	if state.Period != 2 {
		t.Fatalf("expected period 2, got %d", state.Period)
	}
	if state.Phase != MatchPhaseIntermission {
		t.Fatalf("expected intermission phase, got %q", state.Phase)
	}
	if state.PhaseTicks <= 0 {
		t.Fatalf("expected intermission countdown to be active")
	}
}

func TestIntermissionWaitsForUniqueColorsBeforeAutoResume(t *testing.T) {
	state := NewMultiplayerGameState()
	state.Phase = MatchPhaseIntermission
	state.PhaseTicks = 1
	state.HomeColor = TeamColorGreen
	state.AwayColor = TeamColorGreen

	Step(&state, nil)

	if state.Phase != MatchPhaseIntermission {
		t.Fatalf("expected intermission to keep waiting when colors match, got %q", state.Phase)
	}
	if state.PhaseTicks != 0 {
		t.Fatalf("expected countdown to reach zero while waiting for unique colors, got %d", state.PhaseTicks)
	}
}
func TestMultiplayerGameOverStartsPostgameMenu(t *testing.T) {
	state := NewMultiplayerGameState()
	state.Phase = MatchPhasePlaying
	state.PhaseTicks = 0
	state.FaceoffTicks = 0
	state.Period = RegulationPeriods
	state.ClockTicks = 1
	state.Score.Home = 2
	state.Score.Away = 1

	Step(&state, nil)

	if !state.GameOver {
		t.Fatalf("expected game over at end of regulation")
	}
	if state.Phase != MatchPhasePostgame {
		t.Fatalf("expected postgame phase, got %q", state.Phase)
	}
	if state.HomeReady || state.AwayReady {
		t.Fatalf("expected rematch votes to start cleared")
	}
}

func TestPostgameReadyRestartsMultiplayerMatch(t *testing.T) {
	state := NewMultiplayerGameState()
	state.GameOver = true
	state.Phase = MatchPhasePostgame
	state.HomeColor = TeamColorGreen
	state.AwayColor = TeamColorOrange
	state.Score.Home = 4
	state.Score.Away = 3
	state.Period = RegulationPeriods

	Step(&state, []InputFrame{{Team: TeamHome, Ready: true}, {Team: TeamAway, Ready: true}})

	if state.GameOver {
		t.Fatalf("expected rematch votes to restart the game")
	}
	if state.Phase != MatchPhasePlaying {
		t.Fatalf("expected restarted match to be playing, got %q", state.Phase)
	}
	if state.Score.Home != 0 || state.Score.Away != 0 {
		t.Fatalf("expected score reset on rematch, got %+v", state.Score)
	}
	if state.Period != 1 {
		t.Fatalf("expected rematch to restart at period 1, got %d", state.Period)
	}
	if state.HomeColor != TeamColorGreen || state.AwayColor != TeamColorOrange {
		t.Fatalf("expected rematch to preserve colors, got %q and %q", state.HomeColor, state.AwayColor)
	}
	if state.FaceoffTicks <= 0 {
		t.Fatalf("expected rematch to start with a faceoff")
	}
}

func TestSavedShotCountsAsShotOnGoal(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.Puck.Position = Vec2{X: state.AwayGoalie.Position.X - 26, Y: state.AwayGoalie.Position.Y}
	state.Puck.Velocity = Vec2{X: 180, Y: 0}
	state.Puck.ShotTeam = TeamHome
	state.Puck.ShotActive = true

	updatePuck(&state)

	if state.CurrentPeriodStats.Home.ShotsOnGoal != 1 {
		t.Fatalf("expected 1 home shot on goal, got %+v", state.CurrentPeriodStats.Home)
	}
	if state.CurrentPeriodStats.Home.Goals != 0 {
		t.Fatalf("expected no goal on a save, got %+v", state.CurrentPeriodStats.Home)
	}
}

func TestGoalCountsTowardShotsOnGoalAndGoals(t *testing.T) {
	state := NewGameState()
	state.FaceoffTicks = 0
	state.Puck.Position = Vec2{X: AwayGoalLineX - 6, Y: CenterY}
	state.Puck.Velocity = Vec2{X: 900, Y: 0}
	state.Puck.ShotTeam = TeamHome
	state.Puck.ShotActive = true

	updatePuck(&state)

	if state.Score.Home != 1 {
		t.Fatalf("expected home score 1, got %+v", state.Score)
	}
	if state.CurrentPeriodStats.Home.ShotsOnGoal != 1 || state.CurrentPeriodStats.Home.Goals != 1 {
		t.Fatalf("expected 1 shot on goal and 1 goal, got %+v", state.CurrentPeriodStats.Home)
	}
}

func TestIntermissionCapturesCompletedPeriodStats(t *testing.T) {
	state := NewMultiplayerGameState()
	state.Phase = MatchPhasePlaying
	state.PhaseTicks = 0
	state.FaceoffTicks = 0
	state.ClockTicks = 1
	state.CurrentPeriodStats = PeriodStats{
		Period: 1,
		Home:   TeamPeriodStats{ShotsOnGoal: 7, Goals: 2},
		Away:   TeamPeriodStats{ShotsOnGoal: 4, Goals: 1},
	}

	Step(&state, nil)

	if state.LastIntermissionStats.Period != 1 {
		t.Fatalf("expected period 1 summary, got %+v", state.LastIntermissionStats)
	}
	if state.LastIntermissionStats.Home.ShotsOnGoal != 7 || state.LastIntermissionStats.Away.Goals != 1 {
		t.Fatalf("expected completed-period stats to carry into intermission, got %+v", state.LastIntermissionStats)
	}
	if state.CurrentPeriodStats.Period != 2 {
		t.Fatalf("expected current stats to reset for period 2, got %+v", state.CurrentPeriodStats)
	}
	if state.CurrentPeriodStats.Home.ShotsOnGoal != 0 || state.CurrentPeriodStats.Home.Goals != 0 || state.CurrentPeriodStats.Away.ShotsOnGoal != 0 || state.CurrentPeriodStats.Away.Goals != 0 {
		t.Fatalf("expected fresh stats for the next period, got %+v", state.CurrentPeriodStats)
	}
}

func TestSmokeSummaryIncludesCoreState(t *testing.T) {
	summary := SmokeSummary()
	for _, fragment := range []string{"Go Hockey ready.", "tick=", "home=3", "away=3", "puck=("} {
		if !strings.Contains(summary, fragment) {
			t.Fatalf("expected smoke summary to contain %q, got %q", fragment, summary)
		}
	}
}

func TestStatsHelpers(t *testing.T) {
	state := NewGameState()
	recordShotOnGoal(&state, TeamHome)
	recordGoalForTeam(&state, TeamAway)
	registerShotOnGoalIfNeeded(&state, TeamHome)
	if state.CurrentPeriodStats.Home.ShotsOnGoal != 1 || state.CurrentPeriodStats.Away.Goals != 1 {
		t.Fatalf("unexpected stats after direct records: %+v", state.CurrentPeriodStats)
	}

	markShotReleased(&state, TeamAway)
	registerShotOnGoalIfNeeded(&state, TeamAway)
	if state.CurrentPeriodStats.Away.ShotsOnGoal != 1 || !state.Puck.ShotCounted {
		t.Fatalf("expected away shot on goal to register once, got %+v puck=%+v", state.CurrentPeriodStats, state.Puck)
	}
	registerShotOnGoalIfNeeded(&state, TeamAway)
	if state.CurrentPeriodStats.Away.ShotsOnGoal != 1 {
		t.Fatalf("expected shot on goal to count only once, got %+v", state.CurrentPeriodStats)
	}

	clearShotMetadata(&state)
	if state.Puck.ShotTeam != TeamNone || state.Puck.ShotActive || state.Puck.ShotCounted {
		t.Fatalf("expected shot metadata cleared, got %+v", state.Puck)
	}

	finalizePeriodStats(&state, 2)
	if state.LastIntermissionStats.Period != 1 || state.CurrentPeriodStats.Period != 2 {
		t.Fatalf("unexpected period stats after finalize: last=%+v current=%+v", state.LastIntermissionStats, state.CurrentPeriodStats)
	}
}

func TestStatsHelpersIgnoreNilAndTeamNone(t *testing.T) {
	recordShotOnGoal(nil, TeamHome)
	recordGoalForTeam(nil, TeamAway)
	clearShotMetadata(nil)
	markShotReleased(nil, TeamHome)
	registerShotOnGoalIfNeeded(nil, TeamHome)

	state := NewGameState()
	recordShotOnGoal(&state, TeamNone)
	recordGoalForTeam(&state, TeamNone)
	registerShotOnGoalIfNeeded(&state, TeamNone)
	if state.CurrentPeriodStats.Home != (TeamPeriodStats{}) || state.CurrentPeriodStats.Away != (TeamPeriodStats{}) {
		t.Fatalf("expected TeamNone updates to be ignored, got %+v", state.CurrentPeriodStats)
	}
}

func TestMatchHelpers(t *testing.T) {
	state := NewMultiplayerGameState()
	startReadyPhase(&state, MatchPhaseIntermission)
	if state.Phase != MatchPhaseIntermission || state.PhaseTicks <= 0 || state.HomeReady || state.AwayReady {
		t.Fatalf("unexpected ready phase state %+v", state)
	}
	startPlayingPhase(&state)
	if state.Phase != MatchPhasePlaying || state.HomeReady || state.AwayReady {
		t.Fatalf("unexpected playing phase state %+v", state)
	}
	startPostgamePhase(&state)
	if state.Phase != MatchPhasePostgame || state.HomeReady || state.AwayReady {
		t.Fatalf("unexpected postgame phase state %+v", state)
	}
	if got := nextTeamColor(TeamColorBlack, -1); got != TeamColorRed {
		t.Fatalf("expected wrapped previous team color, got %q", got)
	}
	if got := nextTeamColor(TeamColorBlack, 0); got != TeamColorOrange {
		t.Fatalf("expected zero delta to advance, got %q", got)
	}
	if got := otherTeam(TeamHome); got != TeamAway {
		t.Fatalf("expected home opposite away, got %q", got)
	}
	if got := otherTeamColor(&state, TeamHome); got != state.AwayColor {
		t.Fatalf("expected away team color, got %q", got)
	}
	setTeamColor(&state, TeamHome, TeamColorGreen)
	if got := teamColorForTeam(&state, TeamHome); got != TeamColorGreen {
		t.Fatalf("expected updated home team color, got %q", got)
	}
	setTeamReady(&state, TeamAway, true)
	if !teamReady(&state, TeamAway) {
		t.Fatalf("expected away team ready")
	}
	state.HomeColor = TeamColorBlue
	state.AwayColor = TeamColorBlue
	if canReadyTeam(&state, TeamHome) {
		t.Fatalf("expected home ready blocked when away already locked same color")
	}
}
