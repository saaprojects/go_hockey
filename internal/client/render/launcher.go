package render

import (
	"fmt"
	"image/color"

	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type LauncherMenuModel struct {
	SelectedOption int
	SoloColor      sim.TeamColor
	Status         string
	RoomCount      int
}

type JoinBrowserModel struct {
	Rooms        []discovery.Room
	SelectedRoom int
	Status       string
}

type JoinRoomCard struct {
	Index int
	Area  ui.Rect
}

func DrawLauncherMenu(screen *ebiten.Image, model LauncherMenuModel) {
	screen.Fill(colorHUDBackground)
	drawRinkBackdrop(screen)

	topPanelX := sim.CenterX - 290.0
	topPanelY := 42.0
	topPanelWidth := 580.0
	topPanelHeight := 102.0
	footer := LauncherFooterRect()

	topFill := color.RGBA{0xf8, 0xfb, 0xff, 0xff}
	footerFill := color.RGBA{0x13, 0x29, 0x44, 0xff}
	cardFill := color.RGBA{0xfc, 0xfd, 0xff, 0xff}
	selectedFill := color.RGBA{0x1c, 0x42, 0x6b, 0xff}
	mutedText := color.RGBA{0x5b, 0x6c, 0x80, 0xff}
	lightText := color.RGBA{0xee, 0xf5, 0xff, 0xff}
	accentText := color.RGBA{0xa9, 0xd3, 0xff, 0xff}
	inputFill := color.RGBA{0x09, 0x18, 0x2a, 0xff}

	ebitenutil.DrawRect(screen, topPanelX+8, topPanelY+10, topPanelWidth, topPanelHeight, ui.PanelShadowColor)
	ui.DrawRoundedFill(screen, topPanelX, topPanelY, topPanelWidth, topPanelHeight, 24, topFill)
	ui.DrawTextCentered(screen, "Go Hockey", ui.TitleFace(), sim.CenterX, 60, ui.TextDarkColor)
	subtitle := "Choose solo, host, or browse LAN rooms from this same client"
	ui.DrawTextCentered(screen, subtitle, ui.BodyFace(), sim.CenterX, 96, ui.TextDarkColor)
	controls := "Click or press Enter to start. Up or Down changes mode. Esc returns here from a match"
	ui.DrawTextCentered(screen, controls, ui.SmallFace(), sim.CenterX, 122, mutedText)

	labels := []string{"Solo Game", "Host Multiplayer", "Join Multiplayer"}
	details := []string{
		"Play locally against AI in the same client.",
		"Start a local server and advertise it on your LAN.",
		"Browse LAN rooms nearby and click one to connect.",
	}
	for index, label := range labels {
		drawMenuOptionCard(screen, MenuOptionRect(index), label, details[index], model.SelectedOption == index, cardFill, selectedFill)
	}

	ebitenutil.DrawRect(screen, footer.X+8, footer.Y+10, footer.W, footer.H, color.RGBA{0x03, 0x0b, 0x14, 0x44})
	ui.DrawRoundedFill(screen, footer.X, footer.Y, footer.W, footer.H, 20, footerFill)
	colorsLine := "Team colors: Black  |  Orange  |  Green  |  Blue  |  Red"
	ui.DrawTextCentered(screen, colorsLine, ui.BodyFace(), sim.CenterX, footer.Y+18, lightText)
	drawMenuFooter(screen, model, accentText, lightText, inputFill)
	if model.Status != "" {
		ui.DrawTextCentered(screen, model.Status, ui.SmallFace(), sim.CenterX, footer.Y+footer.H+18, ui.TextDarkColor)
	}
}

func DrawJoinBrowser(screen *ebiten.Image, model JoinBrowserModel) {
	screen.Fill(colorHUDBackground)
	drawRinkBackdrop(screen)

	panelX := sim.CenterX - 314.0
	panelY := 54.0
	panelWidth := 628.0
	panelHeight := 560.0
	headFill := color.RGBA{0xf7, 0xfb, 0xff, 0xff}
	bodyFill := color.RGBA{0x12, 0x28, 0x44, 0xff}
	bodyOutline := color.RGBA{0x38, 0x68, 0x99, 0xff}
	lightText := color.RGBA{0xee, 0xf5, 0xff, 0xff}
	accentText := color.RGBA{0xa9, 0xd3, 0xff, 0xff}
	mutedText := color.RGBA{0x6b, 0x7b, 0x8f, 0xff}

	ebitenutil.DrawRect(screen, panelX+10, panelY+12, panelWidth, panelHeight, ui.PanelShadowColor)
	ui.DrawRoundedFill(screen, panelX, panelY, panelWidth, 120, 26, headFill)
	ui.DrawRoundedFill(screen, panelX, panelY+98, panelWidth, panelHeight-98, 26, bodyFill)
	ebitenutil.DrawRect(screen, panelX+26, panelY+140, panelWidth-52, 2, bodyOutline)

	ui.DrawTextCentered(screen, "Join LAN Room", ui.TitleFace(), sim.CenterX, panelY+26, ui.TextDarkColor)
	subtitle := "Nearby hosts appear automatically. Click a room or press Enter to connect."
	ui.DrawTextCentered(screen, subtitle, ui.BodyFace(), sim.CenterX, panelY+62, ui.TextDarkColor)
	help := "Esc returns to the launcher. Up or Down changes rooms while you browse."
	ui.DrawTextCentered(screen, help, ui.SmallFace(), sim.CenterX, panelY+92, mutedText)

	ui.DrawText(screen, "Available Rooms", ui.BodyFace(), panelX+34, panelY+120, accentText)
	if len(model.Rooms) == 0 {
		ui.DrawRoundedFill(screen, panelX+34, panelY+168, panelWidth-68, 122, 18, color.RGBA{0x19, 0x34, 0x55, 0xff})
		ui.DrawTextCentered(screen, "Searching your LAN for open rooms...", ui.BodyFace(), sim.CenterX, panelY+212, lightText)
		secondary := "Have a friend click Host Multiplayer on this same network."
		ui.DrawTextCentered(screen, secondary, ui.SmallFace(), sim.CenterX, panelY+248, accentText)
	} else {
		cursorX, cursorY := ebiten.CursorPosition()
		for _, card := range JoinRoomCards(len(model.Rooms), model.SelectedRoom) {
			room := model.Rooms[card.Index]
			hovered := ui.PointInRect(float64(cursorX), float64(cursorY), card.Area)
			selected := card.Index == model.SelectedRoom
			drawRoomCard(screen, card.Area, room, selected, hovered)
		}
	}

	if model.Status != "" {
		ui.DrawTextCentered(screen, model.Status, ui.SmallFace(), sim.CenterX, panelY+panelHeight-24, lightText)
	}
}

func LauncherFooterRect() ui.Rect {
	return ui.Rect{X: sim.CenterX - 290.0, Y: 536.0, W: 580.0, H: 112.0}
}

func LauncherSoloColorPrevRect() ui.Rect {
	footer := LauncherFooterRect()
	return ui.Rect{X: footer.X + 34, Y: footer.Y + 64, W: 34, H: 30}
}

func LauncherSoloColorLabelRect() ui.Rect {
	footer := LauncherFooterRect()
	return ui.Rect{X: footer.X + 78, Y: footer.Y + 64, W: 136, H: 30}
}

func LauncherSoloColorNextRect() ui.Rect {
	footer := LauncherFooterRect()
	return ui.Rect{X: footer.X + 224, Y: footer.Y + 64, W: 34, H: 30}
}

func MenuOptionRect(index int) ui.Rect {
	cardX := sim.CenterX - 250.0
	cardY := 176.0
	cardWidth := 500.0
	cardHeight := 98.0
	gap := 22.0
	return ui.Rect{X: cardX, Y: cardY + float64(index)*(cardHeight+gap), W: cardWidth, H: cardHeight}
}

func JoinRoomCards(roomCount, roomCursor int) []JoinRoomCard {
	if roomCount == 0 {
		return nil
	}
	const visibleCount = 4
	start := 0
	if roomCursor >= visibleCount {
		start = roomCursor - visibleCount + 1
	}
	if start+visibleCount > roomCount {
		start = roomCount - visibleCount
	}
	if start < 0 {
		start = 0
	}
	cards := []JoinRoomCard{}
	baseX := sim.CenterX - 280.0
	baseY := 226.0
	cardWidth := 560.0
	cardHeight := 72.0
	gap := 16.0
	end := start + visibleCount
	if end > roomCount {
		end = roomCount
	}
	for index := start; index < end; index++ {
		cards = append(cards, JoinRoomCard{
			Index: index,
			Area:  ui.Rect{X: baseX, Y: baseY + float64(index-start)*(cardHeight+gap), W: cardWidth, H: cardHeight},
		})
	}
	return cards
}

func drawRinkBackdrop(screen *ebiten.Image) {
	ui.DrawRoundedFill(screen, sim.RinkLeft-16, sim.RinkTop-16, sim.RinkRight-sim.RinkLeft+32, sim.RinkBottom-sim.RinkTop+32, sim.RinkCornerRadius+16, colorBoard)
	ui.DrawRoundedFill(screen, sim.RinkLeft-10, sim.RinkTop-10, sim.RinkRight-sim.RinkLeft+20, sim.RinkBottom-sim.RinkTop+20, sim.RinkCornerRadius+10, colorBoardOutline)
	ui.DrawRoundedFill(screen, sim.RinkLeft, sim.RinkTop, sim.RinkRight-sim.RinkLeft, sim.RinkBottom-sim.RinkTop, sim.RinkCornerRadius, colorIce)
}

func drawMenuOptionCard(screen *ebiten.Image, area ui.Rect, label, detail string, selected bool, cardFill, selectedFill color.RGBA) {
	fill := cardFill
	stripe := color.RGBA{0x4e, 0x72, 0x97, 0xff}
	titleColor := ui.TextDarkColor
	detailColor := color.RGBA{0x4f, 0x60, 0x74, 0xff}
	shadow := ui.PanelShadowColor
	if selected {
		fill = selectedFill
		stripe = color.RGBA{0x46, 0x9b, 0xff, 0xff}
		titleColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
		detailColor = color.RGBA{0xdd, 0xee, 0xff, 0xff}
		shadow = color.RGBA{0x02, 0x08, 0x11, 0x60}
	}
	ebitenutil.DrawRect(screen, area.X+8, area.Y+10, area.W, area.H, shadow)
	ui.DrawRoundedFill(screen, area.X, area.Y, area.W, area.H, 20, fill)
	ebitenutil.DrawRect(screen, area.X, area.Y, 18, area.H, stripe)
	ui.DrawText(screen, label, ui.BodyFace(), area.X+36, area.Y+24, titleColor)
	ui.DrawText(screen, detail, ui.SmallFace(), area.X+36, area.Y+58, detailColor)
}

func drawMenuFooter(screen *ebiten.Image, model LauncherMenuModel, accentText, lightText, inputFill color.RGBA) {
	footer := LauncherFooterRect()
	titleY := footer.Y + 46
	rowY := footer.Y + 68
	helpY := footer.Y + 94
	switch model.SelectedOption {
	case 0:
		palette := paletteForTeamColor(model.SoloColor)
		cursorX, cursorY := ebiten.CursorPosition()
		ui.DrawText(screen, "Solo team color", ui.SmallFace(), footer.X+34, titleY, accentText)
		prevRect := LauncherSoloColorPrevRect()
		nextRect := LauncherSoloColorNextRect()
		labelRect := LauncherSoloColorLabelRect()
		ui.DrawOverlayButton(screen, prevRect, "<", ui.PointInRect(float64(cursorX), float64(cursorY), prevRect), false)
		ui.DrawRoundedFill(screen, labelRect.X, labelRect.Y, labelRect.W, labelRect.H, 10, inputFill)
		ui.DrawRoundedFill(screen, labelRect.X+10, labelRect.Y+7, 18, 18, 8, palette.Primary)
		ui.DrawText(screen, TeamColorLabel(model.SoloColor), ui.SmallFace(), labelRect.X+40, labelRect.Y+9, lightText)
		ui.DrawOverlayButton(screen, nextRect, ">", ui.PointInRect(float64(cursorX), float64(cursorY), nextRect), false)
		help := "Use Left and Right or click the arrows to change your solo team color"
		ui.DrawText(screen, help, ui.SmallFace(), footer.X+34, helpY, accentText)
	case 1:
		ui.DrawText(screen, "Host Multiplayer", ui.SmallFace(), footer.X+34, titleY, accentText)
		help := "Starts a local server, advertises it on your LAN, and joins from this client"
		ui.DrawText(screen, help, ui.SmallFace(), footer.X+34, rowY+6, lightText)
	case 2:
		ui.DrawText(screen, "Join Multiplayer", ui.SmallFace(), footer.X+34, titleY, accentText)
		ui.DrawRoundedFill(screen, footer.X+34, rowY, footer.W-68, 32, 10, inputFill)
		roomSummary := "No LAN rooms found yet"
		if model.RoomCount == 1 {
			roomSummary = "1 LAN room found nearby"
		}
		if model.RoomCount > 1 {
			roomSummary = fmt.Sprintf("%d LAN rooms found nearby", model.RoomCount)
		}
		ui.DrawText(screen, roomSummary, ui.SmallFace(), footer.X+48, rowY+8, lightText)
		help := "Press Enter to browse rooms, or click Join Multiplayer again"
		ui.DrawText(screen, help, ui.SmallFace(), footer.X+34, helpY, accentText)
	}
}

func drawRoomCard(screen *ebiten.Image, area ui.Rect, room discovery.Room, selected, hovered bool) {
	fill := color.RGBA{0x1a, 0x35, 0x56, 0xff}
	outline := color.RGBA{0x36, 0x63, 0x92, 0xff}
	titleColor := color.RGBA{0xf5, 0xfb, 0xff, 0xff}
	detailColor := color.RGBA{0xb8, 0xd8, 0xf5, 0xff}
	badgeFill := color.RGBA{0x0d, 0x1d, 0x31, 0xff}
	badgeText := color.RGBA{0xb6, 0xd9, 0xff, 0xff}
	if hovered {
		outline = color.RGBA{0x55, 0x92, 0xd1, 0xff}
	}
	if selected {
		fill = color.RGBA{0x25, 0x4d, 0x78, 0xff}
		outline = color.RGBA{0x6f, 0xb4, 0xff, 0xff}
		badgeFill = color.RGBA{0x12, 0x2a, 0x45, 0xff}
	}
	if !room.Joinable() {
		fill = color.RGBA{0x2a, 0x34, 0x41, 0xff}
		outline = color.RGBA{0x5e, 0x67, 0x74, 0xff}
		detailColor = color.RGBA{0xc9, 0xd2, 0xdd, 0xff}
		badgeFill = color.RGBA{0x1b, 0x22, 0x2c, 0xff}
		badgeText = color.RGBA{0xdd, 0xe4, 0xec, 0xff}
	}
	ebitenutil.DrawRect(screen, area.X+6, area.Y+8, area.W, area.H, color.RGBA{0x02, 0x08, 0x11, 0x55})
	ui.DrawRoundedFill(screen, area.X, area.Y, area.W, area.H, 20, fill)
	ebitenutil.DrawRect(screen, area.X, area.Y, area.W, 2, outline)
	ebitenutil.DrawRect(screen, area.X, area.Y+area.H-2, area.W, 2, outline)
	ebitenutil.DrawRect(screen, area.X, area.Y, 2, area.H, outline)
	ebitenutil.DrawRect(screen, area.X+area.W-2, area.Y, 2, area.H, outline)
	ui.DrawText(screen, "Room "+room.Code, ui.BodyFace(), area.X+24, area.Y+22, titleColor)
	hostLabel := truncateLabel(room.Name, 24)
	ui.DrawText(screen, hostLabel, ui.SmallFace(), area.X+24, area.Y+54, detailColor)
	badgeWidth := 112.0
	ui.DrawRoundedFill(screen, area.X+area.W-badgeWidth-18, area.Y+18, badgeWidth, 34, 12, badgeFill)
	playersLabel := fmt.Sprintf("%d/%d players", room.Status.Players, room.Status.Capacity)
	if !room.Joinable() {
		playersLabel = "Room Full"
	}
	playersWidth, _ := ui.MeasureText(playersLabel, ui.SmallFace())
	ui.DrawText(screen, playersLabel, ui.SmallFace(), area.X+area.W-badgeWidth/2-playersWidth/2-18, area.Y+28, badgeText)
}

func truncateLabel(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}
