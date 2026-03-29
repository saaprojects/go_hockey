package sim

import "math"

type goalSegment struct {
	Start Vec2
	End   Vec2
}

func Step(state *GameState, inputs []InputFrame) {
	if state == nil {
		return
	}

	state.Tick++
	homeInput, awayInput := collectTeamInputs(inputs)

	if state.GameOver {
		if state.UseMenus && state.Phase == MatchPhasePostgame {
			updatePostgamePhase(state, homeInput, awayInput)
		}
		return
	}

	if state.UseMenus && state.Phase != MatchPhasePlaying {
		updateMatchPhase(state, homeInput, awayInput)
		return
	}

	if state.GoalPauseTicks > 0 {
		state.GoalPauseTicks--
		updateGoalies(state)
		if state.GoalPauseTicks == 0 {
			setFaceoff(state)
		}
		return
	}

	if state.FaceoffTicks > 0 {
		state.FaceoffTicks--
		updateGoalies(state)
		if state.FaceoffTicks == 0 {
			state.Puck.Velocity = Vec2{X: 40 * state.LastFaceoffDirection}
			state.LastFaceoffDirection *= -1
		}
		return
	}

	if state.Puck.PickupLockTicks > 0 {
		state.Puck.PickupLockTicks--
		if state.Puck.PickupLockTicks == 0 {
			state.Puck.PickupLockTeam = TeamNone
		}
	}

	if homeInput.Switch {
		switchControlToClosest(state, TeamHome)
	}
	if awayInput.Switch {
		switchControlToClosest(state, TeamAway)
	}

	manageUserControl(state, TeamHome)
	manageUserControl(state, TeamAway)
	updateSkaters(state, TeamHome, homeInput)
	updateSkaters(state, TeamAway, awayInput)
	resolveSkaterCollisions(state)
	updateGoalies(state)
	updatePuck(state)
	updateClock(state)
}

func collectTeamInputs(inputs []InputFrame) (TeamInput, TeamInput) {
	var home TeamInput
	var away TeamInput

	for _, input := range inputs {
		normalized := input.Move
		if normalized.Length() > 1.0 {
			normalized = normalized.Normalized()
		}

		switch input.Team {
		case TeamHome:
			home = TeamInput{Active: true, ClientID: input.ClientID, Move: normalized, Shoot: input.Shoot, Pass: input.Pass, Switch: input.Switch, Ready: input.Ready, ColorPrev: input.ColorPrev, ColorNext: input.ColorNext}
		case TeamAway:
			away = TeamInput{Active: true, ClientID: input.ClientID, Move: normalized, Shoot: input.Shoot, Pass: input.Pass, Switch: input.Switch, Ready: input.Ready, ColorPrev: input.ColorPrev, ColorNext: input.ColorNext}
		}
	}

	return home, away
}

func setFaceoff(state *GameState) {
	state.FaceoffTicks = ticksFromSeconds(FaceoffFreeze)
	state.GoalPauseTicks = 0
	state.PuckTrapTicks = 0
	state.Puck.CarrierID = ""
	state.Puck.Velocity = Vec2{}
	state.Puck.Position = Vec2{X: CenterX, Y: CenterY}
	clearShotMetadata(state)
	state.Puck.LastTouchTeam = TeamNone
	state.Puck.PickupLockTeam = TeamNone
	state.Puck.PickupLockTicks = ticksFromSeconds(0.55)

	homePositions := []Vec2{{X: CenterX - 90, Y: CenterY - 105}, {X: CenterX - 56, Y: CenterY}, {X: CenterX - 90, Y: CenterY + 105}}
	awayPositions := []Vec2{{X: CenterX + 90, Y: CenterY - 105}, {X: CenterX + 56, Y: CenterY}, {X: CenterX + 90, Y: CenterY + 105}}

	for index := range state.HomeSkaters {
		state.HomeSkaters[index].Position = homePositions[index]
		state.HomeSkaters[index].Velocity = Vec2{}
		state.HomeSkaters[index].LookDir = Vec2{X: 1, Y: 0}
		state.HomeSkaters[index].ActionCooldownTicks = ticksFromSeconds(0.4)
	}
	for index := range state.AwaySkaters {
		state.AwaySkaters[index].Position = awayPositions[index]
		state.AwaySkaters[index].Velocity = Vec2{}
		state.AwaySkaters[index].LookDir = Vec2{X: -1, Y: 0}
		state.AwaySkaters[index].ActionCooldownTicks = ticksFromSeconds(0.4)
	}

	state.HomeGoalie.Position = Vec2{X: HomeGoalLineX + GoalieOffset, Y: CenterY}
	state.AwayGoalie.Position = Vec2{X: AwayGoalLineX - GoalieOffset, Y: CenterY}
	state.HomeControlled = 1
	state.AwayControlled = 1
}

func attackDir(team Team) float64 {
	if team == TeamHome {
		return 1.0
	}
	return -1.0
}

func goalLineX(team Team) float64 {
	if team == TeamHome {
		return HomeGoalLineX
	}
	return AwayGoalLineX
}

func controlledIndex(state *GameState, team Team) *int {
	if team == TeamHome {
		return &state.HomeControlled
	}
	return &state.AwayControlled
}

func teamSkaters(state *GameState, team Team) *[]SkaterState {
	if team == TeamHome {
		return &state.HomeSkaters
	}
	return &state.AwaySkaters
}

func goalieFor(state *GameState, team Team) *GoalieState {
	if team == TeamHome {
		return &state.HomeGoalie
	}
	return &state.AwayGoalie
}

func carrierForTeam(state *GameState, team Team) *SkaterState {
	if state.Puck.CarrierID == "" {
		return nil
	}
	skater, ok := findSkaterByID(state, state.Puck.CarrierID)
	if !ok || skater.Team != team {
		return nil
	}
	return skater
}

func findSkaterByID(state *GameState, id string) (*SkaterState, bool) {
	for index := range state.HomeSkaters {
		if state.HomeSkaters[index].ID == id {
			return &state.HomeSkaters[index], true
		}
	}
	for index := range state.AwaySkaters {
		if state.AwaySkaters[index].ID == id {
			return &state.AwaySkaters[index], true
		}
	}
	return nil, false
}

func controlledSkater(state *GameState, team Team) *SkaterState {
	skaters := teamSkaters(state, team)
	index := *controlledIndex(state, team)
	if index < 0 || index >= len(*skaters) {
		return nil
	}
	return &(*skaters)[index]
}

func puckFocusPosition(state *GameState) Vec2 {
	if carrier, ok := findSkaterByID(state, state.Puck.CarrierID); ok {
		return carrier.Position
	}
	return state.Puck.Position
}

func switchControlToClosest(state *GameState, team Team) {
	skaters := teamSkaters(state, team)
	if len(*skaters) == 0 {
		return
	}
	target := puckFocusPosition(state)
	bestIndex := 0
	bestDistance := math.MaxFloat64
	for index := range *skaters {
		distance := (*skaters)[index].Position.Sub(target).Length()
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = index
		}
	}
	*controlledIndex(state, team) = bestIndex
}

func manageUserControl(state *GameState, team Team) {
	carrier := carrierForTeam(state, team)
	if carrier == nil {
		return
	}
	skaters := teamSkaters(state, team)
	for index := range *skaters {
		if (*skaters)[index].ID == carrier.ID {
			*controlledIndex(state, team) = index
			return
		}
	}
}

func updateSkaters(state *GameState, team Team, input TeamInput) {
	skaters := teamSkaters(state, team)
	for index := range *skaters {
		skater := &(*skaters)[index]
		if skater.ActionCooldownTicks > 0 {
			skater.ActionCooldownTicks--
		}
		humanControlled := input.Active && index == *controlledIndex(state, team)
		if humanControlled {
			updateControlledSkater(state, skater, input)
		} else {
			updateAISkater(state, skater)
		}
		speedDecay := clamp(1.0-((1.0-skater.Drag)*TickSeconds*9.0), 0.0, 1.0)
		skater.Velocity = skater.Velocity.Mul(speedDecay)
		skater.Position = skater.Position.Add(skater.Velocity.Mul(TickSeconds))
		containSkater(state, skater)
	}
}

func updateControlledSkater(state *GameState, skater *SkaterState, input TeamInput) {
	if skater == nil {
		return
	}
	if skater.ActionCooldownTicks == 0 {
		if input.Pass && state.Puck.CarrierID == skater.ID {
			passPuck(state, skater, input.Move, true)
		} else if input.Shoot {
			if state.Puck.CarrierID == skater.ID {
				shootPuck(state, skater, input.Move, false)
			} else {
				pokeOrBodyCheck(state, skater)
			}
		}
	}
	move := input.Move.Normalized()
	if move.Length() > 0.0 {
		skater.Velocity = skater.Velocity.Add(move.Mul(skater.Acceleration * TickSeconds))
		skater.Velocity = skater.Velocity.Limit(skater.MaxSpeed)
		skater.LookDir = move
	} else if state.Puck.CarrierID == skater.ID {
		skater.Velocity = skater.Velocity.Limit(skater.MaxSpeed * 0.9)
	}
}

func updateAISkater(state *GameState, skater *SkaterState) {
	target := aiTarget(state, skater)
	desired := target.Sub(skater.Position)
	tuning := tuningFor(skater.Team)
	if desired.Length() > 4.0 {
		move := desired.Normalized()
		skater.Velocity = skater.Velocity.Add(move.Mul(skater.Acceleration * TickSeconds * 0.95 * tuning.AIAccel))
		skater.Velocity = skater.Velocity.Limit(skater.MaxSpeed * tuning.AISpeed)
		skater.LookDir = move
	}
	if state.Puck.CarrierID == skater.ID && skater.ActionCooldownTicks == 0 {
		shootingRange := 270.0 * tuning.ShotRange
		if skater.Role == RoleC {
			shootingRange = 315.0 * tuning.ShotRange
		}
		opponent := TeamHome
		if skater.Team == TeamHome {
			opponent = TeamAway
		}
		opponentGoal := Vec2{X: goalLineX(opponent), Y: CenterY}
		distanceToGoal := opponentGoal.Sub(skater.Position).Length()
		closePressure := enemyPressure(state, skater.Team, skater.Position, 68.0)
		hasClearLane := laneToGoalOpen(state, skater)
		inAttackHalf := (skater.Position.X-CenterX)*attackDir(skater.Team) > 0.0
		if distanceToGoal < shootingRange && hasClearLane {
			shootPuck(state, skater, Vec2{}, true)
		} else if closePressure || (inAttackHalf && !hasClearLane) {
			passPuck(state, skater, Vec2{}, false)
		}
	} else if state.Puck.CarrierID != "" && !sameTeamAsCarrier(state, skater.Team) && skater.ActionCooldownTicks == 0 {
		carrier, ok := findSkaterByID(state, state.Puck.CarrierID)
		if ok && carrier.Position.Sub(skater.Position).Length() < 52.0*tuning.CheckRange {
			pokeOrBodyCheck(state, skater)
		}
	} else if state.Puck.CarrierID == "" && state.Puck.Position.Sub(skater.Position).Length() < 44.0 && skater.ActionCooldownTicks == 0 {
		pokeOrBodyCheck(state, skater)
	}
}

func sameTeamAsCarrier(state *GameState, team Team) bool {
	carrier, ok := findSkaterByID(state, state.Puck.CarrierID)
	return ok && carrier.Team == team
}

func enemyPressure(state *GameState, team Team, position Vec2, maxDistance float64) bool {
	enemy := TeamAway
	if team == TeamAway {
		enemy = TeamHome
	}
	skaters := teamSkaters(state, enemy)
	for index := range *skaters {
		if (*skaters)[index].Position.Sub(position).Length() < maxDistance {
			return true
		}
	}
	return false
}

func nearestOpponentDistance(state *GameState, team Team, point Vec2) float64 {
	enemy := TeamAway
	if team == TeamAway {
		enemy = TeamHome
	}
	defenders := teamSkaters(state, enemy)
	best := math.Inf(1)
	for index := range *defenders {
		defender := &(*defenders)[index]
		clearance := defender.Position.Sub(point).Length() - defender.Radius
		if clearance < best {
			best = clearance
		}
	}
	if best == math.Inf(1) {
		return 220.0
	}
	return best
}

func nearestTeammateDistance(state *GameState, team Team, point Vec2, ignoreA, ignoreB string) float64 {
	skaters := teamSkaters(state, team)
	best := math.Inf(1)
	for index := range *skaters {
		mate := &(*skaters)[index]
		if mate.ID == ignoreA || mate.ID == ignoreB {
			continue
		}
		distance := mate.Position.Sub(point).Length()
		if distance < best {
			best = distance
		}
	}
	if best == math.Inf(1) {
		return 220.0
	}
	return best
}

func laneToGoalOpen(state *GameState, skater *SkaterState) bool {
	attackSign := attackDir(skater.Team)
	enemy := TeamAway
	if skater.Team == TeamAway {
		enemy = TeamHome
	}
	defenders := teamSkaters(state, enemy)
	for index := range *defenders {
		defender := &(*defenders)[index]
		if attackSign > 0 && defender.Position.X <= skater.Position.X {
			continue
		}
		if attackSign < 0 && defender.Position.X >= skater.Position.X {
			continue
		}
		if math.Abs(defender.Position.Y-skater.Position.Y) < 46.0 && math.Abs(defender.Position.X-skater.Position.X) < 180.0 {
			return false
		}
	}
	return true
}

func passLaneClearance(state *GameState, attackingTeam Team, from, to Vec2) float64 {
	enemy := TeamAway
	if attackingTeam == TeamAway {
		enemy = TeamHome
	}
	defenders := teamSkaters(state, enemy)
	best := math.Inf(1)
	for index := range *defenders {
		defender := &(*defenders)[index]
		clearance := distanceToSegment(defender.Position, from, to) - defender.Radius
		if clearance < best {
			best = clearance
		}
	}
	if best == math.Inf(1) {
		return 220.0
	}
	return best
}

func clampTargetToRink(target Vec2, marginX, marginY float64) Vec2 {
	return Vec2{
		X: clamp(target.X, RinkLeft+marginX, RinkRight-marginX),
		Y: clamp(target.Y, RinkTop+marginY, RinkBottom-marginY),
	}
}

func roleLaneSign(role Role) float64 {
	switch role {
	case RoleLW:
		return -1.0
	case RoleRW:
		return 1.0
	default:
		return 0.0
	}
}

func isInDefensiveHalf(team Team, x float64) bool {
	return (x-CenterX)*attackDir(team) < 0.0
}

func twoClosestTeammatesToPoint(state *GameState, team Team, point Vec2) (*SkaterState, *SkaterState) {
	skaters := teamSkaters(state, team)
	var first *SkaterState
	var second *SkaterState
	firstDistance := math.Inf(1)
	secondDistance := math.Inf(1)
	for index := range *skaters {
		skater := &(*skaters)[index]
		distance := skater.Position.Sub(point).Length()
		if distance < firstDistance {
			second = first
			secondDistance = firstDistance
			first = skater
			firstDistance = distance
		} else if distance < secondDistance {
			second = skater
			secondDistance = distance
		}
	}
	return first, second
}

func mostDangerousPassTarget(state *GameState, enemyCarrier *SkaterState) *SkaterState {
	if enemyCarrier == nil {
		return nil
	}
	enemyTeam := enemyCarrier.Team
	skaters := teamSkaters(state, enemyTeam)
	bestScore := math.Inf(-1)
	var best *SkaterState
	attack := attackDir(enemyTeam)
	for index := range *skaters {
		mate := &(*skaters)[index]
		if mate.ID == enemyCarrier.ID {
			continue
		}
		progress := (mate.Position.X - enemyCarrier.Position.X) * attack
		lane := passLaneClearance(state, enemyTeam, enemyCarrier.Position, mate.Position)
		openness := nearestOpponentDistance(state, enemyTeam, mate.Position)
		score := progress*0.45 + lane*1.15 + openness*0.75 - math.Abs(mate.Position.Y-enemyCarrier.Position.Y)*0.08
		if score > bestScore {
			best = mate
			bestScore = score
		}
	}
	return best
}

func passLaneCoverTarget(state *GameState, defendingTeam Team, skater *SkaterState, enemyCarrier *SkaterState) Vec2 {
	attack := attackDir(defendingTeam)
	ownGoal := goalieFor(state, defendingTeam).Position
	slotShield := Vec2{
		X: ownGoal.X + attack*185.0,
		Y: clamp(CenterY+(enemyCarrier.Position.Y-CenterY)*0.4+roleLaneSign(skater.Role)*38.0, RinkTop+70.0, RinkBottom-70.0),
	}
	receiver := mostDangerousPassTarget(state, enemyCarrier)
	if receiver == nil {
		return clampTargetToRink(slotShield, 90.0, 55.0)
	}
	intercept := enemyCarrier.Position.Add(receiver.Position.Sub(enemyCarrier.Position).Mul(0.58))
	blended := intercept.Mul(0.62).Add(slotShield.Mul(0.38))
	return clampTargetToRink(blended, 90.0, 55.0)
}

func offensiveSupportTarget(state *GameState, skater *SkaterState, carrier *SkaterState) Vec2 {
	attack := attackDir(skater.Team)
	var candidates []Vec2
	switch skater.Role {
	case RoleLW:
		candidates = []Vec2{{X: attack * 150.0, Y: -150.0}, {X: attack * 110.0, Y: -185.0}, {X: attack * 55.0, Y: -120.0}, {X: attack * 190.0, Y: -72.0}}
	case RoleRW:
		candidates = []Vec2{{X: attack * 150.0, Y: 150.0}, {X: attack * 110.0, Y: 185.0}, {X: attack * 55.0, Y: 120.0}, {X: attack * 190.0, Y: 72.0}}
	default:
		candidates = []Vec2{{X: attack * 110.0, Y: 0.0}, {X: attack * 65.0, Y: -82.0}, {X: attack * 65.0, Y: 82.0}, {X: -attack * 42.0, Y: 0.0}}
	}

	bestTarget := clampTargetToRink(carrier.Position.Add(candidates[0]), 95.0, 50.0)
	bestScore := math.Inf(-1)
	for _, offset := range candidates {
		target := clampTargetToRink(carrier.Position.Add(offset), 95.0, 50.0)
		openness := nearestOpponentDistance(state, skater.Team, target)
		lane := passLaneClearance(state, skater.Team, carrier.Position, target)
		spacing := target.Sub(carrier.Position).Length()
		progress := (target.X - carrier.Position.X) * attack
		laneAffinity := 80.0 - math.Abs(target.Y-skater.LaneY)
		teammateSpacing := nearestTeammateDistance(state, skater.Team, target, skater.ID, carrier.ID)
		score := openness*1.35 + lane*1.10 + progress*0.30 + laneAffinity*0.18 + teammateSpacing*0.15 - math.Abs(spacing-165.0)*0.32
		if skater.Role == RoleC && progress < 30.0 {
			score += 14.0
		}
		if progress < -20.0 {
			score -= 18.0
		}
		if score > bestScore {
			bestScore = score
			bestTarget = target
		}
	}
	return bestTarget
}

func defensiveTarget(state *GameState, skater *SkaterState, enemyCarrier *SkaterState) Vec2 {
	primary, secondary := twoClosestTeammatesToPoint(state, skater.Team, enemyCarrier.Position)
	attack := attackDir(skater.Team)
	if primary != nil && primary.ID == skater.ID {
		offsetY := clamp((skater.LaneY-enemyCarrier.Position.Y)*0.18, -24.0, 24.0)
		pressure := enemyCarrier.Position.Add(Vec2{X: -attack * 28.0, Y: offsetY})
		return clampTargetToRink(pressure, 45.0, 35.0)
	}
	if isInDefensiveHalf(skater.Team, enemyCarrier.Position.X) && secondary != nil && secondary.ID == skater.ID {
		side := roleLaneSign(skater.Role)
		if side == 0.0 {
			if skater.Position.Y <= enemyCarrier.Position.Y {
				side = -1.0
			} else {
				side = 1.0
			}
		}
		support := enemyCarrier.Position.Add(Vec2{X: -attack * 74.0, Y: side * 64.0})
		support.Y = clamp(support.Y+(enemyCarrier.Position.Y-CenterY)*0.12, RinkTop+40.0, RinkBottom-40.0)
		return clampTargetToRink(support, 55.0, 40.0)
	}
	return passLaneCoverTarget(state, skater.Team, skater, enemyCarrier)
}

func aiTarget(state *GameState, skater *SkaterState) Vec2 {
	teamCarrier := carrierForTeam(state, skater.Team)
	enemy := TeamAway
	if skater.Team == TeamAway {
		enemy = TeamHome
	}
	enemyCarrier := carrierForTeam(state, enemy)
	attack := attackDir(skater.Team)
	if state.Puck.CarrierID == skater.ID {
		lanePush := map[Role]float64{RoleLW: -90.0, RoleC: 0.0, RoleRW: 90.0}[skater.Role]
		targetX := clamp(skater.Position.X+attack*175.0, RinkLeft+150.0, RinkRight-150.0)
		targetY := clamp(CenterY+lanePush*0.55+(CenterY-skater.Position.Y)*0.1, RinkTop+40.0, RinkBottom-40.0)
		if math.Abs(targetY-skater.Position.Y) < 18.0 {
			targetY += lanePush * 0.2
		}
		return Vec2{X: targetX, Y: targetY}
	}
	if teamCarrier != nil {
		return offensiveSupportTarget(state, skater, teamCarrier)
	}
	if enemyCarrier != nil {
		return defensiveTarget(state, skater, enemyCarrier)
	}
	predictedPuck := state.Puck.Position.Add(state.Puck.Velocity.Mul(0.18))
	chaser := closestTeammateToPoint(state, skater.Team, predictedPuck)
	if chaser != nil && chaser.ID == skater.ID {
		return clampTargetToRink(predictedPuck, 20.0, 20.0)
	}
	anchorShift := (state.Puck.Position.X - CenterX) * 0.15
	return Vec2{X: clamp(skater.HomeAnchor.X+anchorShift, RinkLeft+100.0, RinkRight-100.0), Y: clamp(skater.LaneY+(state.Puck.Position.Y-CenterY)*0.18, RinkTop+50.0, RinkBottom-50.0)}
}

func closestTeammateToPoint(state *GameState, team Team, point Vec2) *SkaterState {
	skaters := teamSkaters(state, team)
	if len(*skaters) == 0 {
		return nil
	}
	best := &(*skaters)[0]
	bestDistance := best.Position.Sub(point).Length()
	for index := 1; index < len(*skaters); index++ {
		distance := (*skaters)[index].Position.Sub(point).Length()
		if distance < bestDistance {
			best = &(*skaters)[index]
			bestDistance = distance
		}
	}
	return best
}

func bestPassTarget(state *GameState, skater *SkaterState, inputMove Vec2, human bool) (*SkaterState, float64) {
	skaters := teamSkaters(state, skater.Team)
	if len(*skaters) < 2 {
		return nil, math.Inf(-1)
	}
	normalizedInput := inputMove.Normalized()
	attack := attackDir(skater.Team)
	bestScore := math.Inf(-1)
	var bestTarget *SkaterState
	for index := range *skaters {
		mate := &(*skaters)[index]
		if mate.ID == skater.ID {
			continue
		}
		progress := (mate.Position.X - skater.Position.X) * attack
		openness := nearestOpponentDistance(state, skater.Team, mate.Position)
		lane := passLaneClearance(state, skater.Team, skater.Position, mate.Position)
		spacing := mate.Position.Sub(skater.Position).Length()
		score := progress*0.45 + openness*0.90 + lane*1.25 - math.Abs(spacing-150.0)*0.22
		if human && normalizedInput.Length() > 0.4 {
			toMate := mate.Position.Sub(skater.Position).Normalized()
			score += toMate.Dot(normalizedInput) * 140.0
		}
		if lane < 10.0 {
			score -= 45.0
		}
		if score > bestScore {
			bestScore = score
			bestTarget = mate
		}
	}
	return bestTarget, bestScore
}

func shootPuck(state *GameState, skater *SkaterState, inputMove Vec2, ai bool) {
	opponent := TeamHome
	if skater.Team == TeamHome {
		opponent = TeamAway
	}
	opponentGoalie := goalieFor(state, opponent)
	tuning := tuningFor(skater.Team)
	targetX := goalLineX(opponent) + attackDir(skater.Team)*14.0
	goalTargetTop := CenterY - GoalHalfHeight + ShotTargetMargin
	goalTargetBottom := CenterY + GoalHalfHeight - ShotTargetMargin
	goalieBias := CenterY - opponentGoalie.Position.Y
	targetY := clamp(CenterY+goalieBias*0.85, goalTargetTop, goalTargetBottom)
	if !ai && inputMove.Y < -0.2 {
		targetY -= 28.0
	}
	if !ai && inputMove.Y > 0.2 {
		targetY += 28.0
	}
	targetY = clamp(targetY, goalTargetTop, goalTargetBottom)
	shotTarget := Vec2{X: targetX, Y: targetY}
	baseDirection := shotTarget.Sub(skater.Position).Normalized()
	releaseSpeed := PlayerShotSpeed
	if ai {
		releaseSpeed = AIShotSpeed * tuning.ShotSpeed
	}
	releaseSpeed += math.Max(0.0, skater.Velocity.Length()*0.18)
	releasePuck(state, skater, baseDirection.Mul(releaseSpeed), skater.Team, ticksFromSeconds(0.12), true)
	skater.ActionCooldownTicks = ticksFromSeconds(0.65)
}

func passPuck(state *GameState, skater *SkaterState, inputMove Vec2, human bool) {
	bestTarget, bestScore := bestPassTarget(state, skater, inputMove, human)
	if bestTarget == nil {
		return
	}
	if !human && bestScore < 30.0 {
		return
	}
	aim := bestTarget.Position.Sub(skater.Position).Normalized()
	speed := PlayerPassSpeed + bestTarget.Velocity.Length()*0.18
	releasePuck(state, skater, aim.Mul(speed), skater.Team, ticksFromSeconds(0.08), false)
	skater.ActionCooldownTicks = ticksFromSeconds(0.4)
}

func releasePuck(state *GameState, skater *SkaterState, velocity Vec2, lockTeam Team, lockTicks int, isShot bool) {
	state.Puck.CarrierID = ""
	state.Puck.Velocity = velocity
	facing := skater.LookDir.Normalized()
	if facing.Length() < 0.2 {
		facing = Vec2{X: attackDir(skater.Team)}
	}
	state.Puck.Position = skater.Position.Add(facing.Mul(skater.Radius + state.Puck.Radius + 3.0))
	state.Puck.LastTouchTeam = skater.Team
	state.Puck.PickupLockTeam = lockTeam
	state.Puck.PickupLockTicks = lockTicks
	if isShot {
		markShotReleased(state, skater.Team)
	} else {
		clearShotMetadata(state)
	}
}

func pokeOrBodyCheck(state *GameState, skater *SkaterState) {
	enemyTeam := TeamAway
	if skater.Team == TeamAway {
		enemyTeam = TeamHome
	}
	enemyCarrier := carrierForTeam(state, enemyTeam)
	if enemyCarrier != nil {
		displacement := enemyCarrier.Position.Sub(skater.Position)
		if displacement.Length() <= 54.0 {
			knockDir := displacement.Normalized()
			state.Puck.CarrierID = ""
			state.Puck.Position = enemyCarrier.Position.Add(knockDir.Mul(enemyCarrier.Radius + state.Puck.Radius + 4.0))
			state.Puck.Velocity = knockDir.Mul(420.0).Add(skater.LookDir.Normalized().Mul(160.0))
			state.Puck.LastTouchTeam = skater.Team
			state.Puck.PickupLockTeam = enemyTeam
			state.Puck.PickupLockTicks = ticksFromSeconds(0.28)
			clearShotMetadata(state)
			enemyCarrier.Velocity = enemyCarrier.Velocity.Add(knockDir.Mul(90.0))
			skater.ActionCooldownTicks = ticksFromSeconds(0.52)
			return
		}
	}
	if state.Puck.CarrierID == "" && state.Puck.Position.Sub(skater.Position).Length() <= 50.0 {
		burst := skater.LookDir.Normalized().Mul(460.0)
		state.Puck.Velocity = burst
		state.Puck.LastTouchTeam = skater.Team
		state.Puck.PickupLockTeam = skater.Team
		state.Puck.PickupLockTicks = ticksFromSeconds(0.08)
		clearShotMetadata(state)
		skater.ActionCooldownTicks = ticksFromSeconds(0.35)
	}
}

func containSkater(state *GameState, skater *SkaterState) {
	skater.Position = constrainToRink(skater.Position, skater.Radius+4.0)
	combinedNormal := Vec2{}
	framePosition, frameNormal, frameHit := pushCircleOutOfGoalFrames(skater.Position, skater.Radius+2.0, true)
	if frameHit {
		skater.Position = framePosition
		combinedNormal = combinedNormal.Add(frameNormal)
	}
	interiorPosition, interiorNormal, interiorHit := pushCircleOutOfGoalInterior(skater.Position, skater.Radius+2.0)
	if interiorHit {
		skater.Position = interiorPosition
		combinedNormal = combinedNormal.Add(interiorNormal)
	}
	if normal := combinedNormal.Normalized(); normal.Length() > 1e-6 {
		dot := skater.Velocity.Dot(normal)
		if dot < 0.0 {
			skater.Velocity = skater.Velocity.Sub(normal.Mul(dot * 1.1))
		}
	}
	for _, goalie := range []*GoalieState{&state.HomeGoalie, &state.AwayGoalie} {
		displacement := skater.Position.Sub(goalie.Position)
		minimum := skater.Radius + goalie.Radius + 4.0
		if displacement.Length() > 0.0 && displacement.Length() < minimum {
			normal := displacement.Normalized()
			skater.Position = goalie.Position.Add(normal.Mul(minimum))
			skater.Velocity = skater.Velocity.Sub(normal.Mul(skater.Velocity.Dot(normal) * 0.55))
		}
	}
}

func resolveSkaterCollisions(state *GameState) {
	for index := range state.HomeSkaters {
		for other := index + 1; other < len(state.HomeSkaters); other++ {
			separateSkaters(&state.HomeSkaters[index], &state.HomeSkaters[other])
		}
	}
	for index := range state.AwaySkaters {
		for other := index + 1; other < len(state.AwaySkaters); other++ {
			separateSkaters(&state.AwaySkaters[index], &state.AwaySkaters[other])
		}
	}
	for homeIndex := range state.HomeSkaters {
		for awayIndex := range state.AwaySkaters {
			separateSkaters(&state.HomeSkaters[homeIndex], &state.AwaySkaters[awayIndex])
		}
	}
}

func separateSkaters(a, b *SkaterState) {
	if a == nil || b == nil {
		return
	}
	delta := b.Position.Sub(a.Position)
	distance := delta.Length()
	minimum := a.Radius + b.Radius + 2.0
	if distance >= minimum {
		return
	}
	normal := delta.Normalized()
	if normal.Length() < 1e-6 {
		normal = Vec2{X: 1.0}
	}
	push := (minimum - distance) * 0.5
	a.Position = a.Position.Add(normal.Mul(-push))
	b.Position = b.Position.Add(normal.Mul(push))
	a.Velocity = a.Velocity.Add(normal.Mul(-35.0))
	b.Velocity = b.Velocity.Add(normal.Mul(35.0))
}

func updateGoalies(state *GameState) {
	focus := state.Puck.Position
	if carrier, ok := findSkaterByID(state, state.Puck.CarrierID); ok {
		focus = carrier.Position
	}
	for _, goalie := range []*GoalieState{&state.HomeGoalie, &state.AwayGoalie} {
		track := tuningFor(goalie.Team).GoalieTrack
		targetY := focus.Y
		if math.Abs(focus.X-goalie.HomeX) > GoalDepth+34.0 {
			targetY = CenterY + (focus.Y-CenterY)*0.82
		} else {
			targetY = CenterY + (focus.Y-CenterY)*0.48
		}
		goalie.Position.X = goalie.HomeX
		goalie.Position.Y += (clamp(targetY, goalie.MinY, goalie.MaxY) - goalie.Position.Y) * 0.18 * track
		goalie.Position.Y = clamp(goalie.Position.Y, goalie.MinY, goalie.MaxY)
	}
}

func updatePuck(state *GameState) {
	previousPosition := state.Puck.Position
	if carrier, ok := findSkaterByID(state, state.Puck.CarrierID); ok {
		facing := carrier.LookDir.Normalized()
		if facing.Length() < 0.2 {
			facing = Vec2{X: attackDir(carrier.Team)}
		}
		state.Puck.Position = carrier.Position.Add(facing.Mul(carrier.Radius + state.Puck.Radius + 3.0))
		state.Puck.Velocity = carrier.Velocity
		if scoringTeam, scored := checkGoalScored(previousPosition, state.Puck.Position); scored {
			awardGoal(state, scoringTeam)
			return
		}
		state.Puck.Position = keepCarriedPuckOutOfGoalTrap(state.Puck.Position, state.Puck.Radius+1.0)
		updatePuckTrapState(state, previousPosition)
		return
	}

	state.Puck.Position = state.Puck.Position.Add(state.Puck.Velocity.Mul(TickSeconds))
	state.Puck.Velocity = state.Puck.Velocity.Mul(0.992)

	if scoringTeam, scored := checkGoalScored(previousPosition, state.Puck.Position); scored {
		awardGoal(state, scoringTeam)
		return
	}

	state.Puck.Position, state.Puck.Velocity = containPuckToRink(state.Puck.Position, state.Puck.Velocity, state.Puck.Radius+1.0)
	var frameNormal Vec2
	var frameHit bool
	state.Puck.Position, frameNormal, frameHit = pushCircleOutOfGoalFrames(state.Puck.Position, state.Puck.Radius+1.0, false)
	if frameHit {
		dot := state.Puck.Velocity.Dot(frameNormal)
		if dot < 0.0 {
			state.Puck.Velocity = state.Puck.Velocity.Sub(frameNormal.Mul(1.9 * dot)).Mul(0.86)
		}
	}

	for _, goalie := range []*GoalieState{&state.HomeGoalie, &state.AwayGoalie} {
		displacement := state.Puck.Position.Sub(goalie.Position)
		saveRadius := goalie.Radius + state.Puck.Radius + 10.0
		if displacement.Length() <= saveRadius {
			if state.Puck.ShotActive && state.Puck.ShotTeam != TeamNone && state.Puck.ShotTeam != goalie.Team {
				registerShotOnGoalIfNeeded(state, state.Puck.ShotTeam)
			}
			normal := displacement.Normalized()
			if normal.Length() < 1e-6 {
				normal = Vec2{X: attackDir(goalie.Team)}
			}
			state.Puck.Position = goalie.Position.Add(normal.Mul(saveRadius + 1.0))
			dot := state.Puck.Velocity.Dot(normal)
			if dot < 0.0 {
				state.Puck.Velocity = state.Puck.Velocity.Sub(normal.Mul(1.8 * dot)).Mul(0.82)
			} else {
				state.Puck.Velocity = state.Puck.Velocity.Add(normal.Mul(140.0))
			}
			state.Puck.LastTouchTeam = goalie.Team
			state.Puck.PickupLockTeam = goalie.Team
			state.Puck.PickupLockTicks = ticksFromSeconds(0.12)
		}
	}

	if scoringTeam, scored := checkGoalScored(previousPosition, state.Puck.Position); scored {
		awardGoal(state, scoringTeam)
		return
	}

	state.Puck.Position, state.Puck.Velocity = keepLoosePuckOutOfGoalTrap(state.Puck.Position, state.Puck.Velocity, state.Puck.Radius+1.0)

	if state.Puck.PickupLockTicks > 0 {
		updatePuckTrapState(state, previousPosition)
		return
	}
	pickup := findPickupSkater(state)
	if pickup == nil {
		updatePuckTrapState(state, previousPosition)
		return
	}
	state.Puck.CarrierID = pickup.ID
	state.Puck.LastTouchTeam = pickup.Team
	state.Puck.Velocity = Vec2{}
	clearShotMetadata(state)
	facing := pickup.LookDir.Normalized()
	if facing.Length() < 0.2 {
		facing = Vec2{X: attackDir(pickup.Team)}
	}
	state.Puck.Position = pickup.Position.Add(facing.Mul(pickup.Radius + state.Puck.Radius + 3.0))
	if scoringTeam, scored := checkGoalScored(previousPosition, state.Puck.Position); scored {
		awardGoal(state, scoringTeam)
		return
	}
	state.Puck.Position = keepCarriedPuckOutOfGoalTrap(state.Puck.Position, state.Puck.Radius+1.0)
	updatePuckTrapState(state, previousPosition)
}

func findPickupSkater(state *GameState) *SkaterState {
	bestDistance := math.Inf(1)
	var best *SkaterState
	for _, skaters := range [](*[]SkaterState){&state.HomeSkaters, &state.AwaySkaters} {
		for index := range *skaters {
			skater := &(*skaters)[index]
			if state.Puck.PickupLockTeam != TeamNone && skater.Team == state.Puck.PickupLockTeam {
				continue
			}
			distance := skater.Position.Sub(state.Puck.Position).Length()
			pickupRange := skater.Radius + state.Puck.Radius + 10.0
			if distance <= pickupRange && distance < bestDistance {
				bestDistance = distance
				best = skater
			}
		}
	}
	return best
}

func checkGoalScored(previousPosition, position Vec2) (Team, bool) {
	if enteredGoalMouth(previousPosition, position, GoalScorePuckRadius, false) {
		return TeamHome, true
	}
	if enteredGoalMouth(previousPosition, position, GoalScorePuckRadius, true) {
		return TeamAway, true
	}
	return TeamNone, false
}

func enteredGoalMouth(previousPosition, position Vec2, puckRadius float64, leftGoal bool) bool {
	goalX, _, goalTop, goalBottom, _, _ := goalFrameGeometry(leftGoal)
	if positionBehindGoalLine(previousPosition, leftGoal, goalX) {
		return false
	}

	top, bottom := scoringMouthBounds(goalTop, goalBottom, puckRadius)
	if crossedGoalPlane(previousPosition, position, goalX, leftGoal, puckRadius) {
		previousFrontX := goalFrontX(previousPosition, leftGoal, puckRadius)
		currentFrontX := goalFrontX(position, leftGoal, puckRadius)
		deltaX := currentFrontX - previousFrontX
		if math.Abs(deltaX) < 1e-6 {
			return false
		}
		progress := (goalX - previousFrontX) / deltaX
		crossY := previousPosition.Y + (position.Y-previousPosition.Y)*progress
		return crossY >= top && crossY <= bottom
	}

	if !positionOnGoalFace(position, leftGoal, goalX, puckRadius) {
		return false
	}
	if !positionWithinScoringMouth(position, top, bottom) {
		return false
	}
	if !positionOnGoalFace(previousPosition, leftGoal, goalX, puckRadius) {
		return true
	}
	return !positionWithinScoringMouth(previousPosition, top, bottom)
}

func scoringMouthBounds(goalTop, goalBottom, puckRadius float64) (float64, float64) {
	return goalTop - GoalScorePostPad - puckRadius, goalBottom + GoalScorePostPad + puckRadius
}

func crossedGoalPlane(previousPosition, position Vec2, goalX float64, leftGoal bool, puckRadius float64) bool {
	previousFrontX := goalFrontX(previousPosition, leftGoal, puckRadius)
	currentFrontX := goalFrontX(position, leftGoal, puckRadius)
	deltaX := currentFrontX - previousFrontX
	if math.Abs(deltaX) < 1e-6 {
		return false
	}
	if leftGoal {
		if previousFrontX < goalX || currentFrontX > goalX {
			return false
		}
	} else {
		if previousFrontX > goalX || currentFrontX < goalX {
			return false
		}
	}
	progress := (goalX - previousFrontX) / deltaX
	return progress >= 0.0 && progress <= 1.0
}

func goalFrontX(position Vec2, leftGoal bool, puckRadius float64) float64 {
	if leftGoal {
		return position.X - puckRadius
	}
	return position.X + puckRadius
}

func positionOnGoalFace(position Vec2, leftGoal bool, goalX, puckRadius float64) bool {
	if leftGoal {
		return position.X <= goalX+GoalScoreDepthPad+puckRadius
	}
	return position.X >= goalX-GoalScoreDepthPad-puckRadius
}

func positionWithinScoringMouth(position Vec2, top, bottom float64) bool {
	return position.Y >= top && position.Y <= bottom
}

func positionBehindGoalLine(position Vec2, leftGoal bool, goalX float64) bool {
	if leftGoal {
		return position.X < goalX
	}
	return position.X > goalX
}

func awardGoal(state *GameState, team Team) {
	state.PuckTrapTicks = 0
	if state.Puck.ShotActive && state.Puck.ShotTeam == team {
		registerShotOnGoalIfNeeded(state, team)
	}
	recordGoalForTeam(state, team)
	switch team {
	case TeamHome:
		state.Score.Home++
	case TeamAway:
		state.Score.Away++
	default:
		return
	}
	clearShotMetadata(state)
	if state.InOvertime {
		state.GameOver = true
		if state.UseMenus {
			startPostgamePhase(state)
		}
		return
	}
	state.GoalPauseTicks = ticksFromSeconds(goalPauseSecondsForColor(teamColorForTeam(state, team)))
	state.Puck.Velocity = Vec2{}
}

func updateClock(state *GameState) {
	if state == nil || state.GameOver || state.FaceoffTicks > 0 {
		return
	}
	if state.ClockTicks > 0 {
		state.ClockTicks--
	}
	if state.ClockTicks > 0 {
		return
	}
	if state.InOvertime {
		state.GameOver = true
		if state.UseMenus {
			startPostgamePhase(state)
		}
		return
	}
	if state.Period < RegulationPeriods {
		finalizePeriodStats(state, state.Period+1)
		state.Period++
		state.ClockTicks = ticksFromSeconds(PeriodLengthSeconds)
		setFaceoff(state)
		if state.UseMenus {
			startReadyPhase(state, MatchPhaseIntermission)
		}
		return
	}
	if state.Score.Home == state.Score.Away {
		finalizePeriodStats(state, state.Period+1)
		state.InOvertime = true
		state.Period = RegulationPeriods + 1
		state.ClockTicks = ticksFromSeconds(OTLengthSeconds)
		setFaceoff(state)
		if state.UseMenus {
			startReadyPhase(state, MatchPhaseIntermission)
		}
		return
	}
	state.GameOver = true
	if state.UseMenus {
		startPostgamePhase(state)
	}
}

func constrainToRink(position Vec2, radius float64) Vec2 {
	position, _ = containPuckToRink(position, Vec2{}, radius)
	return position
}

func containPuckToRink(position, velocity Vec2, radius float64) (Vec2, Vec2) {
	bounce := 0.86
	minX := RinkLeft + radius
	maxX := RinkRight - radius
	minY := RinkTop + radius
	maxY := RinkBottom - radius

	if position.X < minX {
		position.X = minX
		if velocity.X < 0.0 {
			velocity.X = -velocity.X * bounce
		}
	} else if position.X > maxX {
		position.X = maxX
		if velocity.X > 0.0 {
			velocity.X = -velocity.X * bounce
		}
	}
	if position.Y < minY {
		position.Y = minY
		if velocity.Y < 0.0 {
			velocity.Y = -velocity.Y * bounce
		}
	} else if position.Y > maxY {
		position.Y = maxY
		if velocity.Y > 0.0 {
			velocity.Y = -velocity.Y * bounce
		}
	}

	cornerLimit := RinkCornerRadius - radius
	corners := []struct {
		Center Vec2
		Check  func(Vec2) bool
	}{
		{Center: Vec2{X: RinkLeft + RinkCornerRadius, Y: RinkTop + RinkCornerRadius}, Check: func(p Vec2) bool { return p.X < RinkLeft+RinkCornerRadius && p.Y < RinkTop+RinkCornerRadius }},
		{Center: Vec2{X: RinkRight - RinkCornerRadius, Y: RinkTop + RinkCornerRadius}, Check: func(p Vec2) bool { return p.X > RinkRight-RinkCornerRadius && p.Y < RinkTop+RinkCornerRadius }},
		{Center: Vec2{X: RinkLeft + RinkCornerRadius, Y: RinkBottom - RinkCornerRadius}, Check: func(p Vec2) bool { return p.X < RinkLeft+RinkCornerRadius && p.Y > RinkBottom-RinkCornerRadius }},
		{Center: Vec2{X: RinkRight - RinkCornerRadius, Y: RinkBottom - RinkCornerRadius}, Check: func(p Vec2) bool { return p.X > RinkRight-RinkCornerRadius && p.Y > RinkBottom-RinkCornerRadius }},
	}
	for _, corner := range corners {
		if !corner.Check(position) {
			continue
		}
		displacement := position.Sub(corner.Center)
		distance := displacement.Length()
		if distance <= cornerLimit || distance < 1e-6 {
			continue
		}
		normal := displacement.Normalized()
		position = corner.Center.Add(normal.Mul(cornerLimit))
		dot := velocity.Dot(normal)
		if dot > 0.0 {
			velocity = velocity.Sub(normal.Mul(1.9 * dot)).Mul(bounce)
		}
	}

	return position, velocity
}

func updatePuckTrapState(state *GameState, previousPosition Vec2) {
	if state == nil || state.Puck.CarrierID != "" {
		if state != nil {
			state.PuckTrapTicks = 0
		}
		return
	}

	movement := state.Puck.Position.Sub(previousPosition).Length()
	if puckInGoalTrapZone(state.Puck.Position, state.Puck.Radius+1.0) && movement <= 1.2 && state.Puck.Velocity.Length() <= 48.0 {
		state.PuckTrapTicks++
		if state.PuckTrapTicks >= ticksFromSeconds(GoalTrapFaceoff) {
			setFaceoff(state)
		}
		return
	}

	state.PuckTrapTicks = 0
}

func keepLoosePuckOutOfGoalTrap(position, velocity Vec2, radius float64) (Vec2, Vec2) {
	position, normal, hit := pushCircleOutOfGoalFrames(position, radius, true)
	position, interiorNormal, interiorHit := pushCircleOutOfGoalInterior(position, radius)
	if interiorHit {
		if hit {
			normal = normal.Add(interiorNormal).Normalized()
		} else {
			normal = interiorNormal
		}
		hit = true
	}
	if hit {
		dot := velocity.Dot(normal)
		if dot < 0.0 {
			velocity = velocity.Sub(normal.Mul(1.9 * dot)).Mul(0.86)
		}
	}
	return position, velocity
}

func keepCarriedPuckOutOfGoalTrap(position Vec2, radius float64) Vec2 {
	position, _, _ = pushCircleOutOfGoalFrames(position, radius, true)
	position, _, _ = pushCircleOutOfGoalInterior(position, radius)
	return position
}

func pushCircleOutOfGoalFrames(position Vec2, radius float64, includeFront bool) (Vec2, Vec2, bool) {
	result := position
	combinedNormal := Vec2{}
	hit := false
	for _, leftGoal := range []bool{true, false} {
		for _, segment := range goalFrameSegments(leftGoal, includeFront) {
			next, normal, pushed := pushCircleFromSegment(result, radius, segment.Start, segment.End, 4.0)
			if pushed {
				result = next
				combinedNormal = combinedNormal.Add(normal)
				hit = true
			}
		}
	}
	return result, combinedNormal.Normalized(), hit
}

func pushCircleOutOfGoalInterior(position Vec2, radius float64) (Vec2, Vec2, bool) {
	result := position
	combinedNormal := Vec2{}
	hit := false
	for _, leftGoal := range []bool{true, false} {
		if !pointInsideGoal(result, leftGoal) {
			continue
		}

		bestDistance := math.Inf(1)
		bestPoint := Vec2{}
		bestNormal := Vec2{}
		for _, segment := range goalFrameSegments(leftGoal, true) {
			closest := closestPointOnSegment(result, segment.Start, segment.End)
			normal := closest.Sub(result).Normalized()
			if normal.Length() < 1e-6 {
				normal = outwardGoalSegmentNormal(segment.Start, segment.End, leftGoal)
			}
			distance := result.Sub(closest).Length()
			if distance < bestDistance {
				bestDistance = distance
				bestPoint = closest
				bestNormal = normal
			}
		}

		if bestNormal.Length() < 1e-6 {
			continue
		}
		result = bestPoint.Add(bestNormal.Mul(radius + 4.0))
		combinedNormal = combinedNormal.Add(bestNormal)
		hit = true
	}
	return result, combinedNormal.Normalized(), hit
}

func closestPointOnSegment(position, start, end Vec2) Vec2 {
	segment := end.Sub(start)
	lengthSquared := segment.Dot(segment)
	if lengthSquared < 1e-6 {
		return start
	}
	t := clamp(position.Sub(start).Dot(segment)/lengthSquared, 0.0, 1.0)
	return start.Add(segment.Mul(t))
}

func outwardGoalSegmentNormal(start, end Vec2, leftGoal bool) Vec2 {
	candidate := Vec2{X: -(end.Y - start.Y), Y: end.X - start.X}.Normalized()
	if candidate.Length() < 1e-6 {
		if leftGoal {
			return Vec2{X: -1.0}
		}
		return Vec2{X: 1.0}
	}
	midpoint := start.Add(end).Mul(0.5)
	outside := midpoint.Add(candidate.Mul(6.0))
	if !pointInsideGoal(outside, leftGoal) {
		return candidate
	}
	return candidate.Mul(-1.0)
}

func puckInGoalTrapZone(position Vec2, radius float64) bool {
	for _, leftGoal := range []bool{true, false} {
		goalX, backX, goalTop, goalBottom, _, _ := goalFrameGeometry(leftGoal)
		minX := math.Min(goalX, backX) - radius - 18.0
		maxX := math.Max(goalX, backX) + radius + 18.0
		minY := goalTop - radius - 12.0
		maxY := goalBottom + radius + 12.0
		if position.X < minX || position.X > maxX || position.Y < minY || position.Y > maxY {
			continue
		}
		if pointInsideGoal(position, leftGoal) {
			return true
		}
		for _, segment := range goalFrameSegments(leftGoal, false) {
			if distanceToSegment(position, segment.Start, segment.End) <= radius+10.0 {
				return true
			}
		}
	}
	return false
}

func pushCircleFromSegment(position Vec2, radius float64, start, end Vec2, segmentRadius float64) (Vec2, Vec2, bool) {
	segment := end.Sub(start)
	lengthSquared := segment.Dot(segment)
	var closest Vec2
	if lengthSquared < 1e-6 {
		closest = start
	} else {
		t := clamp(position.Sub(start).Dot(segment)/lengthSquared, 0.0, 1.0)
		closest = start.Add(segment.Mul(t))
	}
	displacement := position.Sub(closest)
	distance := displacement.Length()
	minimum := radius + segmentRadius
	if distance >= minimum {
		return position, Vec2{}, false
	}
	normal := displacement.Normalized()
	if normal.Length() < 1e-6 {
		normal = Vec2{X: -(end.Y - start.Y), Y: end.X - start.X}.Normalized()
		if normal.Length() < 1e-6 {
			normal = Vec2{X: 1.0}
		}
	}
	return closest.Add(normal.Mul(minimum)), normal, true
}

func goalFrameSegments(leftGoal bool, includeFront bool) []goalSegment {
	goalX, backX, goalTop, goalBottom, backTop, backBottom := goalFrameGeometry(leftGoal)
	segments := []goalSegment{
		{Start: Vec2{X: goalX, Y: goalTop}, End: Vec2{X: backX, Y: backTop}},
		{Start: Vec2{X: goalX, Y: goalBottom}, End: Vec2{X: backX, Y: backBottom}},
		{Start: Vec2{X: backX, Y: backTop}, End: Vec2{X: backX, Y: backBottom}},
	}
	if includeFront {
		segments = append(segments, goalSegment{Start: Vec2{X: goalX, Y: goalTop}, End: Vec2{X: goalX, Y: goalBottom}})
	}
	return segments
}

func goalFrameGeometry(leftGoal bool) (goalX, backX, goalTop, goalBottom, backTop, backBottom float64) {
	goalX = HomeGoalLineX
	direction := -1.0
	if !leftGoal {
		goalX = AwayGoalLineX
		direction = 1.0
	}
	goalTop = CenterY - GoalHalfHeight
	goalBottom = CenterY + GoalHalfHeight
	backX = goalX + GoalDepth*direction
	backInset := 24.0
	backTop = goalTop + backInset
	backBottom = goalBottom - backInset
	return goalX, backX, goalTop, goalBottom, backTop, backBottom
}

func pointInsideGoal(position Vec2, leftGoal bool) bool {
	goalX, backX, goalTop, goalBottom, backTop, backBottom := goalFrameGeometry(leftGoal)
	minX := math.Min(goalX, backX)
	maxX := math.Max(goalX, backX)
	if position.X < minX || position.X > maxX {
		return false
	}
	depth := 0.0
	if math.Abs(backX-goalX) > 1e-6 {
		depth = (position.X - goalX) / (backX - goalX)
	}
	top := goalTop + (backTop-goalTop)*depth
	bottom := goalBottom + (backBottom-goalBottom)*depth
	return position.Y >= top+2.0 && position.Y <= bottom-2.0
}
