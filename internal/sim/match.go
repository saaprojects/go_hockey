package sim

type MatchPhase string

type TeamColor string

const (
	MatchPhasePlaying      MatchPhase = "playing"
	MatchPhasePregame      MatchPhase = "pregame"
	MatchPhaseIntermission MatchPhase = "intermission"
	MatchPhasePostgame     MatchPhase = "postgame"
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

func startPostgamePhase(state *GameState) {
	state.Phase = MatchPhasePostgame
	state.PhaseTicks = 0
	state.HomeReady = false
	state.AwayReady = false
}

func updateMatchPhase(state *GameState, homeInput, awayInput TeamInput) {
	applyPhaseInput(state, TeamHome, homeInput)
	applyPhaseInput(state, TeamAway, awayInput)
	if state.HomeReady && state.AwayReady && state.HomeColor != state.AwayColor {
		startPlayingPhase(state)
		return
	}
	if state.Phase != MatchPhaseIntermission {
		return
	}
	if state.PhaseTicks > 0 {
		state.PhaseTicks--
	}
	if state.PhaseTicks == 0 && state.HomeColor != state.AwayColor {
		startPlayingPhase(state)
	}
}

func updatePostgamePhase(state *GameState, homeInput, awayInput TeamInput) {
	applyPostgameInput(state, TeamHome, homeInput)
	applyPostgameInput(state, TeamAway, awayInput)
	if state.HomeReady && state.AwayReady {
		restartMultiplayerMatch(state)
	}
}

func applyPhaseInput(state *GameState, team Team, input TeamInput) {
	if input.ColorPrev {
		setTeamReady(state, team, false)
		setTeamColor(state, team, nextTeamColor(teamColorForTeam(state, team), -1))
	}
	if input.ColorNext {
		setTeamReady(state, team, false)
		setTeamColor(state, team, nextTeamColor(teamColorForTeam(state, team), 1))
	}
	if input.Ready {
		if teamReady(state, team) {
			setTeamReady(state, team, false)
			return
		}
		if canReadyTeam(state, team) {
			setTeamReady(state, team, true)
		}
	}
}

func applyPostgameInput(state *GameState, team Team, input TeamInput) {
	if !input.Ready {
		return
	}
	setTeamReady(state, team, !teamReady(state, team))
}

func restartMultiplayerMatch(state *GameState) {
	homeColor := state.HomeColor
	awayColor := state.AwayColor
	next := NewGameState()
	next.UseMenus = true
	next.HomeColor = homeColor
	next.AwayColor = awayColor
	startPlayingPhase(&next)
	*state = next
}

func nextTeamColor(current TeamColor, delta int) TeamColor {
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
	nextIndex := (currentIndex + delta) % len(teamColorCycle)
	if nextIndex < 0 {
		nextIndex += len(teamColorCycle)
	}
	return teamColorCycle[nextIndex]
}

func canReadyTeam(state *GameState, team Team) bool {
	other := otherTeam(team)
	if !teamReady(state, other) {
		return true
	}
	return teamColorForTeam(state, team) != teamColorForTeam(state, other)
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

func otherTeam(team Team) Team {
	if team == TeamHome {
		return TeamAway
	}
	return TeamHome
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
