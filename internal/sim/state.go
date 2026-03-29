package sim

import "fmt"

type Team string

const (
	TeamNone Team = ""
	TeamHome Team = "home"
	TeamAway Team = "away"
)

type Role string

const (
	RoleLW Role = "LW"
	RoleC  Role = "C"
	RoleRW Role = "RW"
)

type Score struct {
	Home int
	Away int
}

type TeamPeriodStats struct {
	ShotsOnGoal int
	Goals       int
}

type PeriodStats struct {
	Period int
	Home   TeamPeriodStats
	Away   TeamPeriodStats
}

type SkaterState struct {
	ID                  string
	Team                Team
	Role                Role
	LaneY               float64
	HomeAnchor          Vec2
	Position            Vec2
	Velocity            Vec2
	LookDir             Vec2
	Radius              float64
	MaxSpeed            float64
	Acceleration        float64
	Drag                float64
	ActionCooldownTicks int
}

type GoalieState struct {
	Team     Team
	Position Vec2
	HomeX    float64
	MinY     float64
	MaxY     float64
	Radius   float64
}

type PuckState struct {
	Position        Vec2
	Velocity        Vec2
	Radius          float64
	CarrierID       string
	ShotTeam        Team
	ShotActive      bool
	ShotCounted     bool
	LastTouchTeam   Team
	PickupLockTeam  Team
	PickupLockTicks int
}

type InputFrame struct {
	ClientID  string
	Team      Team
	Tick      uint64
	Move      Vec2
	Shoot     bool
	Pass      bool
	Switch    bool
	Ready     bool
	ColorPrev bool
	ColorNext bool
}

type TeamInput struct {
	Active    bool
	ClientID  string
	Move      Vec2
	Shoot     bool
	Pass      bool
	Switch    bool
	Ready     bool
	ColorPrev bool
	ColorNext bool
}

type GameState struct {
	Tick                  uint64
	Score                 Score
	Period                int
	ClockTicks            int
	FaceoffTicks          int
	PhaseTicks            int
	GoalPauseTicks        int
	PuckTrapTicks         int
	InOvertime            bool
	GameOver              bool
	UseMenus              bool
	Phase                 MatchPhase
	HomeReady             bool
	AwayReady             bool
	HomeColor             TeamColor
	AwayColor             TeamColor
	CurrentPeriodStats    PeriodStats
	LastIntermissionStats PeriodStats
	HomeControlled        int
	AwayControlled        int
	LastFaceoffDirection  float64
	HomeSkaters           []SkaterState
	AwaySkaters           []SkaterState
	HomeGoalie            GoalieState
	AwayGoalie            GoalieState
	Puck                  PuckState
}

func NewGameState() GameState {
	homeLanes := []float64{CenterY - 120, CenterY, CenterY + 120}
	awayLanes := []float64{CenterY - 120, CenterY, CenterY + 120}
	roles := []Role{RoleLW, RoleC, RoleRW}

	state := GameState{
		Period:               1,
		ClockTicks:           ticksFromSeconds(PeriodLengthSeconds),
		Phase:                MatchPhasePlaying,
		HomeColor:            TeamColorBlue,
		AwayColor:            TeamColorRed,
		CurrentPeriodStats:   newPeriodStats(1),
		HomeControlled:       1,
		AwayControlled:       1,
		LastFaceoffDirection: 1.0,
		HomeGoalie: GoalieState{
			Team:     TeamHome,
			Position: Vec2{X: HomeGoalLineX + GoalieOffset, Y: CenterY},
			HomeX:    HomeGoalLineX + GoalieOffset,
			MinY:     CenterY - GoalHalfHeight + GoalieReachBuffer,
			MaxY:     CenterY + GoalHalfHeight - GoalieReachBuffer,
			Radius:   23.0,
		},
		AwayGoalie: GoalieState{
			Team:     TeamAway,
			Position: Vec2{X: AwayGoalLineX - GoalieOffset, Y: CenterY},
			HomeX:    AwayGoalLineX - GoalieOffset,
			MinY:     CenterY - GoalHalfHeight + GoalieReachBuffer,
			MaxY:     CenterY + GoalHalfHeight - GoalieReachBuffer,
			Radius:   23.0,
		},
		Puck: PuckState{
			Position: Vec2{X: CenterX, Y: CenterY},
			Radius:   7.0,
		},
	}

	state.HomeSkaters = make([]SkaterState, 0, len(homeLanes))
	state.AwaySkaters = make([]SkaterState, 0, len(awayLanes))

	for index, laneY := range homeLanes {
		anchor := Vec2{X: RinkLeft + 250, Y: laneY}
		state.HomeSkaters = append(state.HomeSkaters, newSkaterState(
			fmt.Sprintf("H%d", index+1),
			TeamHome,
			roles[index],
			laneY,
			anchor,
			Vec2{X: 1, Y: 0},
		))
	}

	for index, laneY := range awayLanes {
		anchor := Vec2{X: RinkRight - 250, Y: laneY}
		state.AwaySkaters = append(state.AwaySkaters, newSkaterState(
			fmt.Sprintf("A%d", index+1),
			TeamAway,
			roles[index],
			laneY,
			anchor,
			Vec2{X: -1, Y: 0},
		))
	}

	setFaceoff(&state)
	return state
}

func newSkaterState(id string, team Team, role Role, laneY float64, anchor, lookDir Vec2) SkaterState {
	return SkaterState{
		ID:           id,
		Team:         team,
		Role:         role,
		LaneY:        laneY,
		HomeAnchor:   anchor,
		Position:     anchor,
		LookDir:      lookDir,
		Radius:       19.0,
		MaxSpeed:     345.0,
		Acceleration: 1260.0,
		Drag:         0.87,
	}
}
