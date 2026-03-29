package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawRoundedFill(screen *ebiten.Image, x, y, width, height, radius float64, fill color.Color) {
	vector.FillRect(screen, float32(x+radius), float32(y), float32(width-radius*2), float32(height), fill, false)
	vector.FillRect(screen, float32(x), float32(y+radius), float32(width), float32(height-radius*2), fill, false)
	vector.FillCircle(screen, float32(x+radius), float32(y+radius), float32(radius), fill, true)
	vector.FillCircle(screen, float32(x+width-radius), float32(y+radius), float32(radius), fill, true)
	vector.FillCircle(screen, float32(x+radius), float32(y+height-radius), float32(radius), fill, true)
	vector.FillCircle(screen, float32(x+width-radius), float32(y+height-radius), float32(radius), fill, true)
}

func DrawLine(screen *ebiten.Image, x1, y1, x2, y2, width float64, clr color.Color) {
	vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), float32(width), clr, true)
}

func InsetRect(area Rect, inset float64) Rect {
	return Rect{X: area.X + inset, Y: area.Y + inset, W: area.W - inset*2, H: area.H - inset*2}
}

func DrawGlow(screen *ebiten.Image, area Rect, radius float64, glow color.RGBA) {
	for index := 0; index < 3; index++ {
		expand := float64(9 - index*3)
		alpha := uint8(int(glow.A) / (index + 2))
		glowArea := Rect{X: area.X - expand, Y: area.Y - expand, W: area.W + expand*2, H: area.H + expand*2}
		DrawRoundedFill(screen, glowArea.X, glowArea.Y, glowArea.W, glowArea.H, radius+expand, WithAlpha(glow, alpha))
	}
}

func DrawRoundedPanel(screen *ebiten.Image, area Rect, radius float64, shadow color.Color, outline color.Color, fill color.Color) {
	DrawRoundedFill(screen, area.X+8, area.Y+10, area.W, area.H, radius, shadow)
	DrawRoundedFill(screen, area.X, area.Y, area.W, area.H, radius, outline)
	inner := InsetRect(area, 3)
	innerRadius := radius - 3
	if innerRadius < 0 {
		innerRadius = 0
	}
	DrawRoundedFill(screen, inner.X, inner.Y, inner.W, inner.H, innerRadius, fill)
}
