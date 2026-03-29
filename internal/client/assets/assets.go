package assets

import "embed"

// SoundFiles stores bundled UI and gameplay audio assets.
//
//go:embed sounds/*.wav
var SoundFiles embed.FS
