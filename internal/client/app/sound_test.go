package app

import (
	"testing"
	"time"

	"hockeyv2/internal/sim"
)

func TestTonePCMAndConcatPCM(t *testing.T) {
	first := tonePCM(440, 660, 80*time.Millisecond, 0.2)
	second := tonePCM(660, 880, 40*time.Millisecond, 0.15)
	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("expected generated PCM clips to be non-empty")
	}
	if len(first)%4 != 0 || len(second)%4 != 0 {
		t.Fatalf("expected stereo 16-bit PCM clips, got %d and %d bytes", len(first), len(second))
	}
	joined := concatPCM(first, second)
	if len(joined) != len(first)+len(second) {
		t.Fatalf("expected concatenated clip length %d, got %d", len(first)+len(second), len(joined))
	}
}

func TestNewSoundboardLoadsGoalSoundsForAllTeamColors(t *testing.T) {
	soundboard := newSoundboard()
	for _, teamColor := range []sim.TeamColor{sim.TeamColorBlack, sim.TeamColorOrange, sim.TeamColorGreen, sim.TeamColorBlue, sim.TeamColorRed} {
		clip := soundboard.goalClips[teamColor]
		if len(clip) == 0 {
			t.Fatalf("expected goal sound clip for %q", teamColor)
		}
	}
}

func TestGoalHornPCMIsLongerThanFaceoffChirp(t *testing.T) {
	goal := goalHornPCM()
	faceoff := tonePCM(560, 430, 85*time.Millisecond, 0.18)
	if len(goal) == 0 || len(faceoff) == 0 {
		t.Fatalf("expected generated sound clips to be non-empty")
	}
	if len(goal) <= len(faceoff) {
		t.Fatalf("expected goal horn to be longer than faceoff chirp, got %d <= %d", len(goal), len(faceoff))
	}
}

func TestPlayMatchStateSoundsIgnoresNilSoundboard(t *testing.T) {
	previous := sim.GameState{Score: sim.Score{Home: 1, Away: 0}, FaceoffTicks: 1}
	current := sim.GameState{Score: sim.Score{Home: 2, Away: 0}}
	playMatchStateSounds(nil, previous, current)
}
