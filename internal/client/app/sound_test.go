package app

import (
	"math/rand"
	"testing"

	"hockeyv2/internal/sim"
)

func TestGoalSoundAssetPathsPreferGenericColorNames(t *testing.T) {
	cases := map[sim.TeamColor]string{
		sim.TeamColorBlack:  "sounds/goal_black.wav",
		sim.TeamColorOrange: "sounds/goal_orange.wav",
		sim.TeamColorGreen:  "sounds/goal_green.wav",
		sim.TeamColorBlue:   "sounds/goal_blue.wav",
		sim.TeamColorRed:    "sounds/goal_red.wav",
	}
	for color, want := range cases {
		paths := goalSoundAssetPaths(color)
		if len(paths) == 0 || paths[0] != want {
			t.Fatalf("expected first asset path for %q to be %q, got %+v", color, want, paths)
		}
	}
}

func TestArenaAmbientAssetPathsPreferGenericName(t *testing.T) {
	paths := arenaAmbientAssetPaths()
	if len(paths) == 0 || paths[0] != "sounds/arena_ambient.mp3" {
		t.Fatalf("expected generic arena ambience path first, got %+v", paths)
	}
}

func TestListMenuMusicAssetPathsFindsBundledTracks(t *testing.T) {
	paths := listMenuMusicAssetPaths()
	if len(paths) == 0 {
		t.Fatalf("expected bundled launcher music tracks")
	}
	for _, assetPath := range paths {
		if !supportedMusicAssetPath(assetPath) {
			t.Fatalf("expected supported music asset path, got %q", assetPath)
		}
	}
}

func TestChooseRandomMenuMusicPathAvoidsImmediateRepeatWhenPossible(t *testing.T) {
	paths := []string{"music/EDM/a.mp3", "music/Emo/b.wav", "music/Metal/c.mp3"}
	random := rand.New(rand.NewSource(1))
	got := chooseRandomMenuMusicPath(paths, random, "music/EDM/a.mp3")
	if got == "music/EDM/a.mp3" {
		t.Fatalf("expected a different track than the previous selection, got %q", got)
	}
}

func TestNewSoundboardLoadsGoalSoundsAndMenuAudio(t *testing.T) {
	soundboard := newSoundboard()
	for _, teamColor := range []sim.TeamColor{sim.TeamColorBlack, sim.TeamColorOrange, sim.TeamColorGreen, sim.TeamColorBlue, sim.TeamColorRed} {
		clip := soundboard.goalClips[teamColor]
		if len(clip) == 0 {
			t.Fatalf("expected goal sound clip for %q", teamColor)
		}
	}
	if len(soundboard.ambientClip) == 0 {
		t.Fatalf("expected arena ambient clip to load")
	}
	if len(soundboard.menuMusicPaths) == 0 {
		t.Fatalf("expected launcher music paths to load")
	}
}

func TestGoalHornPCMIsNonEmptyStereoPCM(t *testing.T) {
	goal := goalHornPCM()
	if len(goal) == 0 {
		t.Fatalf("expected goal horn to be non-empty")
	}
	if len(goal)%4 != 0 {
		t.Fatalf("expected stereo 16-bit PCM clip, got %d bytes", len(goal))
	}
}

func TestPlayMatchStateSoundsIgnoresNilSoundboard(t *testing.T) {
	previous := sim.GameState{Score: sim.Score{Home: 1, Away: 0}, FaceoffTicks: 1}
	current := sim.GameState{Score: sim.Score{Home: 2, Away: 0}}
	playMatchStateSounds(nil, previous, current)
}
