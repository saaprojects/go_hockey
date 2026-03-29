package sim

func newPeriodStats(period int) PeriodStats {
	return PeriodStats{Period: period}
}

func periodStatsForTeam(stats *PeriodStats, team Team) *TeamPeriodStats {
	if team == TeamHome {
		return &stats.Home
	}
	return &stats.Away
}

func recordShotOnGoal(state *GameState, team Team) {
	if state == nil || team == TeamNone {
		return
	}
	periodStatsForTeam(&state.CurrentPeriodStats, team).ShotsOnGoal++
}

func recordGoalForTeam(state *GameState, team Team) {
	if state == nil || team == TeamNone {
		return
	}
	periodStatsForTeam(&state.CurrentPeriodStats, team).Goals++
}

func finalizePeriodStats(state *GameState, nextPeriod int) {
	if state == nil {
		return
	}
	state.LastIntermissionStats = state.CurrentPeriodStats
	state.CurrentPeriodStats = newPeriodStats(nextPeriod)
}

func clearShotMetadata(state *GameState) {
	if state == nil {
		return
	}
	state.Puck.ShotTeam = TeamNone
	state.Puck.ShotActive = false
	state.Puck.ShotCounted = false
}

func markShotReleased(state *GameState, team Team) {
	if state == nil {
		return
	}
	state.Puck.ShotTeam = team
	state.Puck.ShotActive = true
	state.Puck.ShotCounted = false
}

func registerShotOnGoalIfNeeded(state *GameState, team Team) {
	if state == nil || team == TeamNone {
		return
	}
	if !state.Puck.ShotActive || state.Puck.ShotCounted || state.Puck.ShotTeam != team {
		return
	}
	recordShotOnGoal(state, team)
	state.Puck.ShotCounted = true
}
