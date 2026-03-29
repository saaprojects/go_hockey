package ui

import (
	"image/color"

	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func DrawOverlayButton(screen *ebiten.Image, area Rect, label string, hovered bool, primary bool) {
	fill := PanelInsetColor
	outline := PanelStrokeColor
	textColor := TextLightColor
	glow := AccentSoftColor
	if primary {
		fill = AccentDeepColor
		outline = AccentColor
		glow = AccentColor
	}
	if hovered {
		DrawGlow(screen, area, 14, WithAlpha(glow, 84))
		outline = AccentColor
	}
	DrawRoundedPanel(screen, area, 14, PanelShadowColor, outline, fill)
	vector.FillRect(screen, float32(area.X+12), float32(area.Y+10), float32(area.W-24), 1, FrostLineColor, false)
	face := SmallFace()
	if area.H >= 38 {
		face = BodyFace()
	}
	labelWidth, labelHeight := MeasureText(label, face)
	DrawText(screen, label, face, area.X+(area.W-labelWidth)/2, area.Y+(area.H-labelHeight)/2-1, textColor)
}

func ModalMenuPanelRect(entryCount int) Rect {
	height := 182.0 + float64(entryCount-1)*60.0
	return Rect{X: sim.CenterX - 260, Y: sim.CenterY - height/2, W: 520, H: height}
}

func ModalMenuOptionRect(index, entryCount int) Rect {
	panel := ModalMenuPanelRect(entryCount)
	return Rect{X: panel.X + 28, Y: panel.Y + 106 + float64(index)*60, W: panel.W - 56, H: 46}
}

func DrawModalMenu(screen *ebiten.Image, title, subtitle, footer string, entries []MenuEntry, selected int) {
	if len(entries) == 0 {
		return
	}
	DrawRoundedFill(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, 0, OverlayColor)
	panel := ModalMenuPanelRect(len(entries))
	DrawGlow(screen, panel, 26, WithAlpha(AccentSoftColor, 54))
	DrawRoundedPanel(screen, panel, 26, PanelShadowColor, PanelStrokeColor, PanelColor)
	header := Rect{X: panel.X + 22, Y: panel.Y + 18, W: panel.W - 44, H: 54}
	DrawRoundedPanel(screen, header, 18, color.RGBA{0, 0, 0, 0}, WithAlpha(PanelStrokeBrightColor, 180), PanelAltColor)
	DrawTextCentered(screen, title, HeadingFace(), sim.CenterX, header.Y+12, TextLightColor)
	DrawTextCentered(screen, subtitle, BodyFace(), sim.CenterX, panel.Y+78, TextSoftColor)
	cursorX, cursorY := ebiten.CursorPosition()
	for index, entry := range entries {
		area := ModalMenuOptionRect(index, len(entries))
		hovered := PointInRect(float64(cursorX), float64(cursorY), area)
		drawModalMenuButton(screen, area, entry.Label, index == selected, hovered, entry.Disabled)
	}
	if footer != "" {
		DrawTextCentered(screen, footer, SmallFace(), sim.CenterX, panel.Y+panel.H-28, TextMutedColor)
	}
}

func drawModalMenuButton(screen *ebiten.Image, area Rect, label string, selected, hovered, disabled bool) {
	fill := PanelInsetColor
	outline := PanelStrokeColor
	textColor := TextLightColor
	if disabled {
		fill = color.RGBA{0x19, 0x26, 0x36, 0xff}
		outline = color.RGBA{0x3f, 0x52, 0x68, 0xff}
		textColor = DisabledTextColor
	} else if selected {
		fill = AccentDeepColor
		outline = AccentColor
	} else if hovered {
		outline = PanelStrokeBrightColor
	}
	if hovered && !disabled {
		DrawGlow(screen, area, 16, WithAlpha(AccentSoftColor, 68))
	}
	DrawRoundedPanel(screen, area, 15, PanelShadowColor, outline, fill)
	DrawTextCentered(screen, label, BodyFace(), area.X+area.W/2, area.Y+10, textColor)
}
