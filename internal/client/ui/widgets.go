package ui

import (
	"image/color"

	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func DrawOverlayButton(screen *ebiten.Image, area Rect, label string, hovered bool, primary bool) {
	fill := color.RGBA{0xe6, 0xec, 0xf4, 0xff}
	outline := color.RGBA{0xbb, 0xca, 0xd8, 0xff}
	textColor := TextDarkColor
	if primary {
		fill = color.RGBA{0x13, 0x3d, 0x68, 0xff}
		outline = color.RGBA{0x2f, 0x7b, 0xc4, 0xff}
		textColor = color.RGBA{0xf2, 0xf8, 0xff, 0xff}
	}
	if hovered {
		outline = color.RGBA{0x56, 0x9f, 0xed, 0xff}
	}
	DrawRoundedFill(screen, area.X, area.Y, area.W, area.H, 12, fill)
	ebitenutil.DrawRect(screen, area.X, area.Y, area.W, 2, outline)
	ebitenutil.DrawRect(screen, area.X, area.Y+area.H-2, area.W, 2, outline)
	ebitenutil.DrawRect(screen, area.X, area.Y, 2, area.H, outline)
	ebitenutil.DrawRect(screen, area.X+area.W-2, area.Y, 2, area.H, outline)
	labelWidth, _ := MeasureText(label, SmallFace())
	DrawText(screen, label, SmallFace(), area.X+(area.W-labelWidth)/2, area.Y+9, textColor)
}

func ModalMenuPanelRect(entryCount int) Rect {
	height := 164.0 + float64(entryCount-1)*58.0
	return Rect{X: sim.CenterX - 230, Y: sim.CenterY - height/2, W: 460, H: height}
}

func ModalMenuOptionRect(index, entryCount int) Rect {
	panel := ModalMenuPanelRect(entryCount)
	return Rect{X: panel.X + 28, Y: panel.Y + 90 + float64(index)*58, W: panel.W - 56, H: 44}
}

func DrawModalMenu(screen *ebiten.Image, title, subtitle, footer string, entries []MenuEntry, selected int) {
	if len(entries) == 0 {
		return
	}
	ebitenutil.DrawRect(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, OverlayColor)
	panel := ModalMenuPanelRect(len(entries))
	ebitenutil.DrawRect(screen, panel.X+8, panel.Y+10, panel.W, panel.H, PanelShadowColor)
	DrawRoundedFill(screen, panel.X, panel.Y, panel.W, panel.H, 24, PanelColor)
	DrawTextCentered(screen, title, TitleFace(), sim.CenterX, panel.Y+24, TextDarkColor)
	DrawTextCentered(screen, subtitle, BodyFace(), sim.CenterX, panel.Y+60, TextDarkColor)
	cursorX, cursorY := ebiten.CursorPosition()
	for index, entry := range entries {
		area := ModalMenuOptionRect(index, len(entries))
		hovered := PointInRect(float64(cursorX), float64(cursorY), area)
		drawModalMenuButton(screen, area, entry.Label, index == selected, hovered, entry.Disabled)
	}
	if footer != "" {
		DrawTextCentered(screen, footer, SmallFace(), sim.CenterX, panel.Y+panel.H-28, color.RGBA{0x5b, 0x6c, 0x80, 0xff})
	}
}

func drawModalMenuButton(screen *ebiten.Image, area Rect, label string, selected, hovered, disabled bool) {
	fill := color.RGBA{0xe6, 0xec, 0xf4, 0xff}
	outline := color.RGBA{0xb8, 0xc6, 0xd7, 0xff}
	textColor := TextDarkColor
	if disabled {
		fill = color.RGBA{0xdb, 0xe2, 0xea, 0xff}
		outline = color.RGBA{0xc4, 0xcf, 0xdb, 0xff}
		textColor = color.RGBA{0x7b, 0x89, 0x97, 0xff}
	} else if selected {
		fill = color.RGBA{0x13, 0x3d, 0x68, 0xff}
		outline = color.RGBA{0x56, 0x9f, 0xed, 0xff}
		textColor = color.RGBA{0xf2, 0xf8, 0xff, 0xff}
	} else if hovered {
		outline = color.RGBA{0x6f, 0x95, 0xbe, 0xff}
	}
	DrawRoundedFill(screen, area.X, area.Y, area.W, area.H, 12, fill)
	ebitenutil.DrawRect(screen, area.X, area.Y, area.W, 2, outline)
	ebitenutil.DrawRect(screen, area.X, area.Y+area.H-2, area.W, 2, outline)
	ebitenutil.DrawRect(screen, area.X, area.Y, 2, area.H, outline)
	ebitenutil.DrawRect(screen, area.X+area.W-2, area.Y, 2, area.H, outline)
	DrawTextCentered(screen, label, BodyFace(), area.X+area.W/2, area.Y+11, textColor)
}
