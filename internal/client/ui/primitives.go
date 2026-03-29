package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawRoundedFill(screen *ebiten.Image, x, y, width, height, radius float64, fill color.Color) {
	ebitenutil.DrawRect(screen, x+radius, y, width-radius*2, height, fill)
	ebitenutil.DrawRect(screen, x, y+radius, width, height-radius*2, fill)
	vector.DrawFilledCircle(screen, float32(x+radius), float32(y+radius), float32(radius), fill, true)
	vector.DrawFilledCircle(screen, float32(x+width-radius), float32(y+radius), float32(radius), fill, true)
	vector.DrawFilledCircle(screen, float32(x+radius), float32(y+height-radius), float32(radius), fill, true)
	vector.DrawFilledCircle(screen, float32(x+width-radius), float32(y+height-radius), float32(radius), fill, true)
}

func DrawLine(screen *ebiten.Image, x1, y1, x2, y2, width float64, clr color.Color) {
	vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), float32(width), clr, true)
}
