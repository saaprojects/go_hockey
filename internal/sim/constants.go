package sim

const (
	TickRate    = 60
	TickSeconds = 1.0 / 60.0
)

const (
	WindowWidth  = 1280.0
	WindowHeight = 820.0
	RinkLeft     = 110.0
	RinkTop      = 110.0
	RinkRight    = 1170.0
	RinkBottom   = 700.0

	RinkCornerRadius = 78.0
	GoalHalfHeight   = 88.0
	GoalDepth        = 40.0
	GoalLineOffset   = 88.0
	GoalieOffset     = 18.0

	HomeGoalLineX = RinkLeft + GoalLineOffset
	AwayGoalLineX = RinkRight - GoalLineOffset
	CenterX       = (RinkLeft + RinkRight) / 2.0
	CenterY       = (RinkTop + RinkBottom) / 2.0
)

const (
	RegulationPeriods   = 3
	PeriodLengthSeconds = 120.0
	OTLengthSeconds     = 60.0
	FaceoffFreeze       = 2.4
)

const (
	PlayerPassSpeed   = 540.0
	PlayerShotSpeed   = PlayerPassSpeed * 2.0
	AIShotSpeed       = PlayerPassSpeed * 1.85
	GoalieReachBuffer = 28.0
	ShotTargetMargin  = 10.0
)

type TeamTuning struct {
	AISpeed     float64
	AIAccel     float64
	ShotSpeed   float64
	ShotRange   float64
	CheckRange  float64
	GoalieTrack float64
}

var teamTuning = map[Team]TeamTuning{
	TeamHome: {
		AISpeed:     1.0,
		AIAccel:     1.0,
		ShotSpeed:   1.0,
		ShotRange:   1.0,
		CheckRange:  1.0,
		GoalieTrack: 0.92,
	},
	TeamAway: {
		AISpeed:     0.9,
		AIAccel:     0.84,
		ShotSpeed:   0.9,
		ShotRange:   0.82,
		CheckRange:  0.82,
		GoalieTrack: 0.72,
	},
}

func tuningFor(team Team) TeamTuning {
	tuning, ok := teamTuning[team]
	if !ok {
		return teamTuning[TeamHome]
	}
	return tuning
}

func ticksFromSeconds(seconds float64) int {
	return int(seconds*TickRate + 0.5)
}
