package netcode

import (
	"testing"

	"hockeyv2/internal/sim"
)

func TestMessageFromInputRoundTrip(t *testing.T) {
	input := sim.InputFrame{
		ClientID:  "client-1",
		Team:      sim.TeamAway,
		Tick:      42,
		Move:      sim.Vec2{X: 1, Y: -2},
		Shoot:     true,
		Pass:      true,
		Switch:    true,
		Ready:     true,
		ColorPrev: true,
		ColorNext: true,
	}

	message := MessageFromInput(input, "client-override")
	if message.Kind != MessageInputFrame {
		t.Fatalf("expected input frame kind, got %q", message.Kind)
	}
	if message.ClientID != "client-override" {
		t.Fatalf("expected overridden client id, got %q", message.ClientID)
	}

	roundTrip := message.ToInputFrame()
	if roundTrip.ClientID != "client-override" || roundTrip.Team != input.Team || roundTrip.Tick != input.Tick {
		t.Fatalf("unexpected round-trip input frame: %+v", roundTrip)
	}
	if !roundTrip.Shoot || !roundTrip.Pass || !roundTrip.Switch || !roundTrip.Ready || !roundTrip.ColorPrev || !roundTrip.ColorNext {
		t.Fatalf("expected boolean fields to survive round trip, got %+v", roundTrip)
	}
}
