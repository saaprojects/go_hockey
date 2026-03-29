package sim

type MatchPhase string

type TeamColor string

const (
	MatchPhasePlaying      MatchPhase = "playing"
	MatchPhasePregame      MatchPhase = "pregame"
	MatchPhaseIntermission MatchPhase = "intermission"
)

const (
	TeamColorBlack  TeamColor = "black"
	TeamColorOrange TeamColor = "orange"
	TeamColorGreen  TeamColor = "green"
	TeamColorBlue   TeamColor = "blue"
	TeamColorRed    TeamColor = "red"
)

const MenuCountdownSeconds = 10.0

var teamColorCycle = []TeamColor{
	TeamColorBlack,
	TeamColorOrange,
	TeamColorGreen,
	TeamColorBlue,
	TeamColorRed,
}

func NewMultiplayerGameState() GameState {
	state := NewGameState()
	state.UseMenus = true
	startReadyPhase(&state, MatchPhasePregame)
	return state
}

func startReadyPhase(state *GameState, phase MatchPhase) {
	state.Phase = phase
	state.PhaseTicks = 0
	if phase == MatchPhaseIntermission {
		state.PhaseTicks = ticksFromSeconds(MenuCountdownSeconds)
	}
	state.HomeReady = false
	state.AwayReady = false
}

func startPlayingPhase(state *GameState) {
	state.Phase = MatchPhasePlaying
	state.PhaseTicks = 0
	state.HomeReady = false
	state.AwayReady = false
}

func updateMatchPhase(state *GameState, homeInput, awayInput TeamInput) {
	applyPhaseInput(state, TeamHome, homeInput)
	applyPhaseInput(state, TeamAway, awayInput)
	if state.HomeReady && state.AwayReady {
		startPlayingPhase(state)
		return
	}
	if state.Phase != MatchPhaseIntermission {
		return
	}
	if state.PhaseTicks > 0 {
		state.PhaseTicks--
	}
	if state.PhaseTicks == 0 {
		startPlayingPhase(state)
	}
}

func applyPhaseInput(state *GameState, team Team, input TeamInput) {
	if input.ColorPrev {
		setTeamReady(state, team, false)
		setTeamColor(state, team, nextAvailableTeamColor(teamColorForTeam(state, team), otherTeamColor(state, team), -1))
	}
	if input.ColorNext {
		setTeamReady(state, team, false)
		setTeamColor(state, team, nextAvailableTeamColor(teamColorForTeam(state, team), otherTeamColor(state, team), 1))
	}
	if input.Ready {
		setTeamReady(state, team, !teamReady(state, team))
	}
}

func nextAvailableTeamColor(current, blocked TeamColor, delta int) TeamColor {
	if delta == 0 {
		delta = 1
	}
	currentIndex := 0
	for index, candidate := range teamColorCycle {
		if candidate == current {
			currentIndex = index
			break
		}
	}
	for step := 1; step <= len(teamColorCycle); step++ {
		nextIndex := (currentIndex + step*delta) % len(teamColorCycle)
		if nextIndex < 0 {
			nextIndex += len(teamColorCycle)
		}
		candidate := teamColorCycle[nextIndex]
		if candidate != blocked {
			return candidate
		}
	}
	return current
}

func teamColorForTeam(state *GameState, team Team) TeamColor {
	if team == TeamHome {
		return state.HomeColor
	}
	return state.AwayColor
}

func otherTeamColor(state *GameState, team Team) TeamColor {
	if team == TeamHome {
		return state.AwayColor
	}
	return state.HomeColor
}

func setTeamColor(state *GameState, team Team, color TeamColor) {
	if team == TeamHome {
		state.HomeColor = color
		return
	}
	state.AwayColor = color
}

func teamReady(state *GameState, team Team) bool {
	if team == TeamHome {
		return state.HomeReady
	}
	return state.AwayReady
}

func setTeamReady(state *GameState, team Team, ready bool) {
	if team == TeamHome {
		state.HomeReady = ready
		return
	}
	state.AwayReady = ready
}
