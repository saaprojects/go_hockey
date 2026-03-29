package client

import (
	"image/color"

	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type matchMenuMode int

type matchMenuAction int

type matchMenuEntry struct {
	Label    string
	Disabled bool
}

type matchMenuState struct {
	Mode     matchMenuMode
	Selected int
}

const (
	matchMenuModeHidden matchMenuMode = iota
	matchMenuModePause
	matchMenuModePostgame
	matchMenuModeDisconnected
)

const (
	matchMenuActionNone matchMenuAction = iota
	matchMenuActionQuit
	matchMenuActionRoomMenu
)

func (m matchMenuState) Visible() bool {
	return m.Mode != matchMenuModeHidden
}

func (m *matchMenuState) Open(mode matchMenuMode) {
	m.Mode = mode
	m.Selected = 0
}

func (m *matchMenuState) Close() {
	m.Mode = matchMenuModeHidden
	m.Selected = 0
}

func updateMatchMenuSelection(menu *matchMenuState, entries []matchMenuEntry) (int, bool) {
	if menu == nil || len(entries) == 0 {
		return 0, false
	}
	menu.Selected = clampMatchMenuSelection(menu.Selected, entries)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if index, ok := matchMenuOptionAtCursor(len(entries)); ok {
			menu.Selected = index
			if !entries[index].Disabled {
				return index, true
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		menu.Selected = nextMatchMenuSelection(menu.Selected, -1, entries)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		menu.Selected = nextMatchMenuSelection(menu.Selected, 1, entries)
	}
	if (inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) && !entries[menu.Selected].Disabled {
		return menu.Selected, true
	}
	return 0, false
}

func clampMatchMenuSelection(selected int, entries []matchMenuEntry) int {
	if len(entries) == 0 {
		return 0
	}
	if selected < 0 || selected >= len(entries) || entries[selected].Disabled {
		for index, entry := range entries {
			if !entry.Disabled {
				return index
			}
		}
		return 0
	}
	return selected
}

func nextMatchMenuSelection(selected, delta int, entries []matchMenuEntry) int {
	if len(entries) == 0 {
		return 0
	}
	selected = clampMatchMenuSelection(selected, entries)
	for step := 0; step < len(entries); step++ {
		selected = (selected + delta + len(entries)) % len(entries)
		if !entries[selected].Disabled {
			return selected
		}
	}
	return clampMatchMenuSelection(selected, entries)
}

func drawMatchMenuOverlay(screen *ebiten.Image, title, subtitle, footer string, entries []matchMenuEntry, selected int) {
	if len(entries) == 0 {
		return
	}
	selected = clampMatchMenuSelection(selected, entries)
	ebitenutil.DrawRect(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, colorOverlay)
	panel := matchMenuPanelRect(len(entries))
	ebitenutil.DrawRect(screen, panel.x+8, panel.y+10, panel.w, panel.h, colorPanelShadow)
	drawRoundedFill(screen, panel.x, panel.y, panel.w, panel.h, 24, colorPanel)
	drawUITextCentered(screen, title, uiTitleFace, sim.CenterX, panel.y+24, colorTextDark)
	drawUITextCentered(screen, subtitle, uiBodyFace, sim.CenterX, panel.y+60, colorTextDark)
	cursorX, cursorY := ebiten.CursorPosition()
	for index, entry := range entries {
		area := matchMenuOptionRect(index, len(entries))
		hovered := pointInRect(float64(cursorX), float64(cursorY), area)
		drawMatchMenuButton(screen, area, entry.Label, index == selected, hovered, entry.Disabled)
	}
	if footer != "" {
		drawUITextCentered(screen, footer, uiSmallFace, sim.CenterX, panel.y+panel.h-28, color.RGBA{0x5b, 0x6c, 0x80, 0xff})
	}
}

func drawMatchMenuButton(screen *ebiten.Image, area rect, label string, selected, hovered, disabled bool) {
	fill := color.RGBA{0xe6, 0xec, 0xf4, 0xff}
	outline := color.RGBA{0xb8, 0xc6, 0xd7, 0xff}
	textColor := colorTextDark
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
	drawRoundedFill(screen, area.x, area.y, area.w, area.h, 12, fill)
	ebitenutil.DrawRect(screen, area.x, area.y, area.w, 2, outline)
	ebitenutil.DrawRect(screen, area.x, area.y+area.h-2, area.w, 2, outline)
	ebitenutil.DrawRect(screen, area.x, area.y, 2, area.h, outline)
	ebitenutil.DrawRect(screen, area.x+area.w-2, area.y, 2, area.h, outline)
	drawUITextCentered(screen, label, uiBodyFace, area.x+area.w/2, area.y+11, textColor)
}

func matchMenuPanelRect(entryCount int) rect {
	height := 164.0 + float64(entryCount-1)*58.0
	return rect{x: sim.CenterX - 230, y: sim.CenterY - height/2, w: 460, h: height}
}

func matchMenuOptionRect(index, entryCount int) rect {
	panel := matchMenuPanelRect(entryCount)
	return rect{x: panel.x + 28, y: panel.y + 90 + float64(index)*58, w: panel.w - 56, h: 44}
}

func matchMenuOptionAtCursor(entryCount int) (int, bool) {
	cursorX, cursorY := ebiten.CursorPosition()
	for index := 0; index < entryCount; index++ {
		if pointInRect(float64(cursorX), float64(cursorY), matchMenuOptionRect(index, entryCount)) {
			return index, true
		}
	}
	return 0, false
}
