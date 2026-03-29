package client

import (
	"image/color"

	"hockeyv2/internal/sim"
)

type teamPalette struct {
	Primary color.RGBA
	Trim    color.RGBA
}

func paletteForTeamColor(teamColor sim.TeamColor) teamPalette {
	switch teamColor {
	case sim.TeamColorBlack:
		return teamPalette{Primary: color.RGBA{0x18, 0x1b, 0x20, 0xff}, Trim: color.RGBA{0xe7, 0xec, 0xf2, 0xff}}
	case sim.TeamColorOrange:
		return teamPalette{Primary: color.RGBA{0xec, 0x74, 0x21, 0xff}, Trim: color.RGBA{0xff, 0xf1, 0xe6, 0xff}}
	case sim.TeamColorGreen:
		return teamPalette{Primary: color.RGBA{0x1f, 0x9d, 0x55, 0xff}, Trim: color.RGBA{0xe8, 0xff, 0xf2, 0xff}}
	case sim.TeamColorRed:
		return teamPalette{Primary: color.RGBA{0xd6, 0x3b, 0x2d, 0xff}, Trim: color.RGBA{0xff, 0xf6, 0xf4, 0xff}}
	default:
		return teamPalette{Primary: color.RGBA{0x1f, 0x7a, 0xe0, 0xff}, Trim: color.RGBA{0xf7, 0xf9, 0xff, 0xff}}
	}
}

func paletteForTeam(state sim.GameState, team sim.Team) teamPalette {
	if team == sim.TeamHome {
		return paletteForTeamColor(state.HomeColor)
	}
	return paletteForTeamColor(state.AwayColor)
}

func teamColorLabel(teamColor sim.TeamColor) string {
	switch teamColor {
	case sim.TeamColorBlack:
		return "Black"
	case sim.TeamColorOrange:
		return "Orange"
	case sim.TeamColorGreen:
		return "Green"
	case sim.TeamColorRed:
		return "Red"
	default:
		return "Blue"
	}
}
