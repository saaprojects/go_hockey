package assets

import "embed"

// SoundFiles stores bundled UI and gameplay audio assets.
//
//go:embed sounds/*.wav sounds/*.mp3
var SoundFiles embed.FS

// MusicFiles stores bundled launcher music assets.
//
//go:embed music/*/*.wav music/*/*.mp3
var MusicFiles embed.FS
