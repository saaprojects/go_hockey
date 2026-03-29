package sim

import "fmt"

func SmokeSummary() string {
	state := NewGameState()
	Step(&state, nil)
	return fmt.Sprintf(
		"Go Hockey ready. tick=%d home=%d away=%d faceoff=%d puck=(%.0f, %.0f)\n",
		state.Tick,
		len(state.HomeSkaters),
		len(state.AwaySkaters),
		state.FaceoffTicks,
		state.Puck.Position.X,
		state.Puck.Position.Y,
	)
}
