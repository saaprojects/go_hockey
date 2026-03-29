package render

import (
	"fmt"
	"image/color"
	"strings"

	"hockeyv2/internal/client/ui"
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type LauncherMenuModel struct {
	SelectedOption int
	Status         string
	RoomCount      int
}

type LaunchSetupModel struct {
	ModeLabel    string
	Description  string
	ConfirmLabel string
	Color        sim.TeamColor
	Status       string
}

type JoinBrowserModel struct {
	Rooms        []discovery.Room
	SelectedRoom int
	Status       string
}

type OnlineRoomModel struct {
	RoomName     string
	RoomCode     string
	FocusedField int
	Status       string
}

type JoinRoomCard struct {
	Index int
	Area  ui.Rect
}

func DrawLauncherMenu(screen *ebiten.Image, model LauncherMenuModel) {
	screen.Fill(colorHUDBackground)
	drawRinkBackdrop(screen)

	hero := ui.Rect{X: sim.CenterX - 270, Y: 26, W: 540, H: 78}
	ui.DrawGlow(screen, hero, 24, ui.WithAlpha(ui.AccentSoftColor, 66))
	ui.DrawRoundedPanel(screen, hero, 24, ui.PanelShadowColor, ui.PanelStrokeBrightColor, ui.PanelColor)
	ui.DrawTextCentered(screen, "GO HOCKEY", ui.DisplayFace(), sim.CenterX, hero.Y+24, ui.TextLightColor)
	ui.DrawTextCentered(screen, "Arcade-style hockey for solo, LAN, or internet play", ui.BodyFace(), sim.CenterX, 128, ui.TextDarkColor)
	ui.DrawTextCentered(screen, "Click a mode card to continue. Keyboard still works if you want it.", ui.SmallFace(), sim.CenterX, 152, ui.TextMutedColor)

	cursorX, cursorY := ebiten.CursorPosition()
	labels := []string{"Solo Game", "Host Multiplayer", "Join Multiplayer", "Online Rooms"}
	details := []string{
		"Play against AI locally with one keyboard.",
		"Start a LAN server and jump in from this client.",
		"Browse rooms on your network and join with one click.",
		"Create a named room or join by 5-character room code.",
	}
	for index, label := range labels {
		area := MenuOptionRect(index, len(labels))
		hovered := ui.PointInRect(float64(cursorX), float64(cursorY), area)
		drawMenuOptionCard(screen, area, index, label, details[index], model.SelectedOption == index, hovered)
	}

	drawLauncherStatusBar(screen, model)
}

func DrawLaunchSetup(screen *ebiten.Image, model LaunchSetupModel) {
	ui.DrawRoundedFill(screen, 0, 0, sim.WindowWidth, sim.WindowHeight, 0, ui.OverlayColor)
	panel := LaunchSetupPanelRect()
	ui.DrawGlow(screen, panel, 26, ui.WithAlpha(ui.AccentSoftColor, 58))
	ui.DrawRoundedPanel(screen, panel, 28, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)

	header := ui.Rect{X: panel.X + 24, Y: panel.Y + 18, W: panel.W - 48, H: 52}
	ui.DrawRoundedPanel(screen, header, 18, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeBrightColor, 180), ui.PanelAltColor)
	ui.DrawTextCentered(screen, model.ModeLabel, ui.HeadingFace(), sim.CenterX, header.Y+12, ui.TextLightColor)
	ui.DrawTextCentered(screen, model.Description, ui.BodyFace(), sim.CenterX, panel.Y+94, ui.TextSoftColor)

	cursorX, cursorY := ebiten.CursorPosition()
	for index, teamColor := range launchSetupColors() {
		chip := LaunchSetupColorChipRect(index)
		hovered := ui.PointInRect(float64(cursorX), float64(cursorY), chip)
		selected := model.Color == teamColor
		drawColorChip(screen, chip, teamColor, selected, hovered)
	}

	ui.DrawTextCentered(screen, fmt.Sprintf("You: %s   |   Opponent: %s", TeamColorLabel(model.Color), TeamColorLabel(launchSetupOpponentColor(model.Color))), ui.BodyFace(), sim.CenterX, panel.Y+182, ui.TextSoftColor)

	backRect := LaunchSetupBackRect()
	confirmRect := LaunchSetupConfirmRect()
	ui.DrawOverlayButton(screen, backRect, "Back", ui.PointInRect(float64(cursorX), float64(cursorY), backRect), false)
	ui.DrawOverlayButton(screen, confirmRect, model.ConfirmLabel, ui.PointInRect(float64(cursorX), float64(cursorY), confirmRect), true)

	footer := model.Status
	if footer == "" {
		footer = "Left and Right change color. Enter confirms. Esc goes back."
	}
	ui.DrawTextCentered(screen, footer, ui.SmallFace(), sim.CenterX, panel.Y+panel.H-22, ui.TextMutedColor)
}

func DrawJoinBrowser(screen *ebiten.Image, model JoinBrowserModel) {
	screen.Fill(colorHUDBackground)
	drawRinkBackdrop(screen)

	panel := ui.Rect{X: sim.CenterX - 370, Y: 52, W: 740, H: 596}
	ui.DrawGlow(screen, panel, 28, ui.WithAlpha(ui.AccentSoftColor, 54))
	ui.DrawRoundedPanel(screen, panel, 30, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)

	header := ui.Rect{X: panel.X + 26, Y: panel.Y + 20, W: panel.W - 52, H: 72}
	ui.DrawRoundedPanel(screen, header, 22, color.RGBA{0, 0, 0, 0}, ui.PanelStrokeBrightColor, ui.PanelAltColor)
	ui.DrawTextCentered(screen, "JOIN LAN ROOM", ui.DisplayFace(), sim.CenterX, header.Y+22, ui.TextLightColor)
	ui.DrawTextCentered(screen, "Nearby hosts appear automatically. Pick one and jump in.", ui.BodyFace(), sim.CenterX, panel.Y+108, ui.TextSoftColor)
	ui.DrawTextCentered(screen, "Esc goes back. Up and Down change selection if you prefer the keyboard.", ui.SmallFace(), sim.CenterX, panel.Y+134, ui.TextMutedColor)

	section := ui.Rect{X: panel.X + 28, Y: panel.Y + 160, W: panel.W - 56, H: 390}
	ui.DrawRoundedPanel(screen, section, 22, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 190), ui.PanelInsetColor)
	ui.DrawText(screen, "Available Rooms", ui.BodyFace(), section.X+18, section.Y+16, ui.TextSoftColor)
	ui.DrawLine(screen, section.X+18, section.Y+46, section.X+section.W-18, section.Y+46, 1, ui.FrostLineColor)

	if len(model.Rooms) == 0 {
		empty := ui.Rect{X: section.X + 28, Y: section.Y + 86, W: section.W - 56, H: 164}
		ui.DrawRoundedPanel(screen, empty, 22, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 180), ui.PanelAltColor)
		ui.DrawTextCentered(screen, "Searching your LAN for open rooms...", ui.HeadingFace(), sim.CenterX, empty.Y+48, ui.TextLightColor)
		ui.DrawTextCentered(screen, "Have a friend click Host Multiplayer on the same network.", ui.BodyFace(), sim.CenterX, empty.Y+94, ui.TextSoftColor)
	} else {
		cursorX, cursorY := ebiten.CursorPosition()
		for _, card := range JoinRoomCards(len(model.Rooms), model.SelectedRoom) {
			room := model.Rooms[card.Index]
			hovered := ui.PointInRect(float64(cursorX), float64(cursorY), card.Area)
			selected := card.Index == model.SelectedRoom
			drawRoomCard(screen, card.Area, room, selected, hovered)
		}
	}

	footer := ui.Rect{X: panel.X + 28, Y: panel.Y + panel.H - 74, W: panel.W - 56, H: 38}
	ui.DrawRoundedPanel(screen, footer, 18, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 170), ui.PanelInsetColor)
	status := model.Status
	if status == "" {
		status = "Choose a room to connect"
	}
	ui.DrawTextCentered(screen, status, ui.SmallFace(), sim.CenterX, footer.Y+10, ui.TextSoftColor)
}

func DrawOnlineRoom(screen *ebiten.Image, model OnlineRoomModel) {
	screen.Fill(colorHUDBackground)
	drawRinkBackdrop(screen)

	panel := OnlineRoomPanelRect()
	ui.DrawGlow(screen, panel, 28, ui.WithAlpha(ui.AccentSoftColor, 60))
	ui.DrawRoundedPanel(screen, panel, 30, ui.PanelShadowColor, ui.PanelStrokeColor, ui.PanelColor)

	header := ui.Rect{X: panel.X + 28, Y: panel.Y + 20, W: panel.W - 56, H: 64}
	ui.DrawRoundedPanel(screen, header, 22, color.RGBA{0, 0, 0, 0}, ui.PanelStrokeBrightColor, ui.PanelAltColor)
	ui.DrawTextCentered(screen, "ONLINE ROOMS", ui.DisplayFace(), sim.CenterX, header.Y+18, ui.TextLightColor)
	ui.DrawTextCentered(screen, "Create a room, share the code, or join with the code a friend gives you.", ui.BodyFace(), sim.CenterX, panel.Y+104, ui.TextSoftColor)

	cursorX, cursorY := ebiten.CursorPosition()
	createFocused := model.FocusedField == 0
	joinFocused := model.FocusedField == 1

	createCard := OnlineRoomCreateCardRect()
	drawOnlineRoomCard(screen, createCard, "Create Online Room", "Start a room, become the host, and we will generate a code for your friend.", createFocused)
	ui.DrawInputField(screen, OnlineRoomNameFieldRect(), "Room Name", model.RoomName, "Weekend Session", createFocused)
	createButton := OnlineRoomCreateButtonRect()
	ui.DrawOverlayButton(screen, createButton, "Create Room", ui.PointInRect(float64(cursorX), float64(cursorY), createButton), true)

	joinCard := OnlineRoomJoinCardRect()
	drawOnlineRoomCard(screen, joinCard, "Join by Room Code", "Enter the 5-character code and jump straight into that room.", joinFocused)
	ui.DrawInputField(screen, OnlineRoomCodeFieldRect(), "Room Code", model.RoomCode, "ABCDE", joinFocused)
	joinButton := OnlineRoomJoinButtonRect()
	ui.DrawOverlayButton(screen, joinButton, "Join Room", ui.PointInRect(float64(cursorX), float64(cursorY), joinButton), true)

	backRect := OnlineRoomBackRect()
	ui.DrawOverlayButton(screen, backRect, "Back", ui.PointInRect(float64(cursorX), float64(cursorY), backRect), false)

	statusRect := OnlineRoomStatusRect()
	ui.DrawRoundedPanel(screen, statusRect, 18, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 170), ui.PanelInsetColor)
	status := model.Status
	if status == "" {
		status = "Tab switches fields. Enter creates or joins. Esc goes back."
	}
	ui.DrawTextCentered(screen, status, ui.SmallFace(), statusRect.X+statusRect.W/2, statusRect.Y+11, ui.TextSoftColor)
}

func LaunchSetupPanelRect() ui.Rect {
	return ui.Rect{X: sim.CenterX - 336, Y: 192, W: 672, H: 300}
}

func LaunchSetupColorChipRect(index int) ui.Rect {
	panel := LaunchSetupPanelRect()
	return ui.Rect{X: panel.X + 32 + float64(index)*122, Y: panel.Y + 128, W: 110, H: 38}
}

func LaunchSetupBackRect() ui.Rect {
	panel := LaunchSetupPanelRect()
	return ui.Rect{X: panel.X + 32, Y: panel.Y + 220, W: 150, H: 40}
}

func LaunchSetupConfirmRect() ui.Rect {
	panel := LaunchSetupPanelRect()
	return ui.Rect{X: panel.X + panel.W - 182, Y: panel.Y + 220, W: 150, H: 40}
}

func OnlineRoomPanelRect() ui.Rect {
	return ui.Rect{X: sim.CenterX - 352, Y: 58, W: 704, H: 548}
}

func OnlineRoomCreateCardRect() ui.Rect {
	panel := OnlineRoomPanelRect()
	return ui.Rect{X: panel.X + 28, Y: panel.Y + 142, W: panel.W - 56, H: 150}
}

func OnlineRoomJoinCardRect() ui.Rect {
	panel := OnlineRoomPanelRect()
	return ui.Rect{X: panel.X + 28, Y: panel.Y + 308, W: panel.W - 56, H: 150}
}

func OnlineRoomNameFieldRect() ui.Rect {
	card := OnlineRoomCreateCardRect()
	return ui.Rect{X: card.X + 20, Y: card.Y + 64, W: card.W - 218, H: 70}
}

func OnlineRoomCodeFieldRect() ui.Rect {
	card := OnlineRoomJoinCardRect()
	return ui.Rect{X: card.X + 20, Y: card.Y + 64, W: card.W - 218, H: 70}
}

func OnlineRoomCreateButtonRect() ui.Rect {
	card := OnlineRoomCreateCardRect()
	return ui.Rect{X: card.X + card.W - 168, Y: card.Y + 85, W: 144, H: 44}
}

func OnlineRoomJoinButtonRect() ui.Rect {
	card := OnlineRoomJoinCardRect()
	return ui.Rect{X: card.X + card.W - 168, Y: card.Y + 85, W: 144, H: 44}
}

func OnlineRoomBackRect() ui.Rect {
	panel := OnlineRoomPanelRect()
	return ui.Rect{X: panel.X + 28, Y: panel.Y + panel.H - 70, W: 140, H: 42}
}

func OnlineRoomStatusRect() ui.Rect {
	panel := OnlineRoomPanelRect()
	back := OnlineRoomBackRect()
	return ui.Rect{X: back.X + back.W + 18, Y: back.Y, W: panel.X + panel.W - (back.X + back.W + 18) - 28, H: 42}
}

func MenuOptionRect(index, optionCount int) ui.Rect {
	cardX := sim.CenterX - 308.0
	cardWidth := 616.0
	cardHeight := 78.0
	gap := 14.0
	blockHeight := float64(optionCount)*cardHeight + float64(optionCount-1)*gap
	cardY := 176.0 + (372.0-blockHeight)/2
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
	baseX := sim.CenterX - 322.0
	baseY := 250.0
	cardWidth := 644.0
	cardHeight := 74.0
	gap := 14.0
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
	ui.DrawRoundedFill(screen, sim.RinkLeft-18, sim.RinkTop-18, sim.RinkRight-sim.RinkLeft+36, sim.RinkBottom-sim.RinkTop+36, sim.RinkCornerRadius+18, color.RGBA{0xf7, 0xfa, 0xfd, 0xff})
	ui.DrawRoundedFill(screen, sim.RinkLeft-11, sim.RinkTop-11, sim.RinkRight-sim.RinkLeft+22, sim.RinkBottom-sim.RinkTop+22, sim.RinkCornerRadius+11, color.RGBA{0x98, 0xac, 0xc2, 0xff})
	ui.DrawRoundedFill(screen, sim.RinkLeft, sim.RinkTop, sim.RinkRight-sim.RinkLeft, sim.RinkBottom-sim.RinkTop, sim.RinkCornerRadius, color.RGBA{0xd6, 0xe8, 0xf5, 0xff})
	ui.DrawRoundedFill(screen, sim.RinkLeft+18, sim.RinkTop+18, sim.RinkRight-sim.RinkLeft-36, sim.RinkBottom-sim.RinkTop-36, sim.RinkCornerRadius-18, color.RGBA{0xe5, 0xf2, 0xfb, 0x74})

	ui.DrawLine(screen, sim.CenterX, sim.RinkTop+18, sim.CenterX, sim.RinkBottom-18, 4, color.RGBA{0xd9, 0x97, 0xa8, 0x66})
	ui.DrawLine(screen, sim.RinkLeft+240, sim.RinkTop+18, sim.RinkLeft+240, sim.RinkBottom-18, 5, color.RGBA{0x62, 0xa3, 0xef, 0x68})
	ui.DrawLine(screen, sim.RinkRight-240, sim.RinkTop+18, sim.RinkRight-240, sim.RinkBottom-18, 5, color.RGBA{0x62, 0xa3, 0xef, 0x68})

	vector.StrokeCircle(screen, float32(sim.CenterX), float32(sim.CenterY), 88, 3, color.RGBA{0xd6, 0x7d, 0x7d, 0x4c}, true)
	for _, circleX := range []float64{sim.RinkLeft + 180, sim.RinkRight - 180} {
		for _, circleY := range []float64{sim.CenterY - 140, sim.CenterY + 140} {
			vector.StrokeCircle(screen, float32(circleX), float32(circleY), 60, 2, color.RGBA{0xd6, 0x7d, 0x7d, 0x38}, true)
		}
	}

	vector.FillRect(screen, float32(sim.RinkLeft+28), float32(sim.RinkTop+24), float32(sim.RinkRight-sim.RinkLeft-56), 8, color.RGBA{0xff, 0xff, 0xff, 0x24}, false)
	vector.FillRect(screen, float32(sim.RinkLeft+28), float32(sim.RinkBottom-32), float32(sim.RinkRight-sim.RinkLeft-56), 8, color.RGBA{0xb8, 0xd9, 0xf3, 0x1c}, false)
}

func drawMenuOptionCard(screen *ebiten.Image, area ui.Rect, option int, label, detail string, selected, hovered bool) {
	fill := ui.PanelColor
	outline := ui.PanelStrokeColor
	titleColor := ui.TextLightColor
	detailColor := ui.TextSoftColor
	if hovered {
		outline = ui.PanelStrokeBrightColor
	}
	if selected {
		fill = ui.PanelAltColor
		outline = ui.AccentColor
		ui.DrawGlow(screen, area, 22, ui.WithAlpha(ui.AccentSoftColor, 70))
	}
	ui.DrawRoundedPanel(screen, area, 24, ui.PanelShadowColor, outline, fill)

	iconArea := ui.Rect{X: area.X + 18, Y: area.Y + 10, W: 62, H: 58}
	drawModeIcon(screen, iconArea, option, selected || hovered)
	ui.DrawText(screen, label, ui.HeadingFace(), area.X+96, area.Y+15, titleColor)
	ui.DrawText(screen, detail, ui.BodyFace(), area.X+96, area.Y+46, detailColor)
}

func drawModeIcon(screen *ebiten.Image, area ui.Rect, option int, active bool) {
	iconColor := ui.TextSoftColor
	iconOutline := ui.PanelStrokeBrightColor
	if active {
		iconColor = ui.TextLightColor
		iconOutline = ui.AccentColor
	}
	cx := float32(area.X + area.W/2)
	cy := float32(area.Y + area.H/2)
	vector.FillCircle(screen, cx, cy, 28, ui.WithAlpha(ui.PanelInsetColor, 240), true)
	vector.StrokeCircle(screen, cx, cy, 28, 3, iconOutline, true)
	switch option {
	case 0:
		vector.FillCircle(screen, cx-6, cy+4, 16, color.RGBA{0x13, 0x16, 0x1b, 0xff}, true)
		vector.StrokeCircle(screen, cx-6, cy+4, 16, 3, iconColor, true)
		vector.FillRect(screen, cx-18, cy+14, 24, 3, ui.WithAlpha(iconColor, 170), false)
	case 1:
		drawCrowdIcon(screen, cx, cy, iconColor)
		ui.DrawLine(screen, float64(cx+16), float64(cy-18), float64(cx+16), float64(cy-6), 3, ui.AccentColor)
		ui.DrawLine(screen, float64(cx+10), float64(cy-12), float64(cx+22), float64(cy-12), 3, ui.AccentColor)
	case 2:
		drawCrowdIcon(screen, cx, cy, iconColor)
		ui.DrawLine(screen, float64(cx+8), float64(cy+12), float64(cx+24), float64(cy+12), 3, ui.AccentColor)
		ui.DrawLine(screen, float64(cx+16), float64(cy+6), float64(cx+24), float64(cy+12), 3, ui.AccentColor)
		ui.DrawLine(screen, float64(cx+16), float64(cy+18), float64(cx+24), float64(cy+12), 3, ui.AccentColor)
	default:
		vector.FillCircle(screen, cx-12, cy, 6, iconColor, true)
		vector.FillCircle(screen, cx+12, cy-10, 6, iconColor, true)
		vector.FillCircle(screen, cx+12, cy+10, 6, iconColor, true)
		ui.DrawLine(screen, float64(cx-6), float64(cy), float64(cx+6), float64(cy-8), 3, ui.AccentColor)
		ui.DrawLine(screen, float64(cx-6), float64(cy), float64(cx+6), float64(cy+8), 3, ui.AccentColor)
	}
}

func drawCrowdIcon(screen *ebiten.Image, cx, cy float32, clr color.Color) {
	vector.FillCircle(screen, cx-10, cy-6, 9, clr, true)
	vector.FillCircle(screen, cx+8, cy-8, 8, clr, true)
	vector.FillCircle(screen, cx+18, cy+3, 6, ui.WithAlpha(ui.AccentColor, 210), true)
	vector.FillRect(screen, cx-20, cy+6, 24, 8, clr, false)
	vector.FillRect(screen, cx+1, cy+4, 18, 8, clr, false)
}

func drawLauncherStatusBar(screen *ebiten.Image, model LauncherMenuModel) {
	footer := ui.Rect{X: sim.CenterX - 322, Y: 568, W: 644, H: 42}
	ui.DrawRoundedPanel(screen, footer, 18, color.RGBA{0, 0, 0, 0}, ui.WithAlpha(ui.PanelStrokeColor, 170), ui.PanelInsetColor)
	ui.DrawTextCentered(screen, launcherStatus(model), ui.SmallFace(), sim.CenterX, footer.Y+12, ui.TextSoftColor)
}

func launcherStatus(model LauncherMenuModel) string {
	if model.Status != "" {
		return model.Status
	}
	switch model.SelectedOption {
	case 0:
		return "Choose Solo Game to pick your color and start a local match against the computer."
	case 1:
		return "Choose Host Multiplayer to pick your color and open a LAN room."
	case 2:
		if model.RoomCount == 1 {
			return "1 LAN room is available now. Choose Join Multiplayer to browse it."
		}
		if model.RoomCount > 1 {
			return fmt.Sprintf("%d LAN rooms are available now. Choose Join Multiplayer to browse them.", model.RoomCount)
		}
		return "Choose Join Multiplayer to browse LAN rooms on your network."
	default:
		return "Choose Online Rooms to create a room, host it, or join one by code."
	}
}

func drawOnlineRoomCard(screen *ebiten.Image, area ui.Rect, title, detail string, focused bool) {
	outline := ui.WithAlpha(ui.PanelStrokeColor, 190)
	fill := ui.PanelInsetColor
	if focused {
		outline = ui.AccentColor
		fill = ui.PanelAltColor
		ui.DrawGlow(screen, area, 18, ui.WithAlpha(ui.AccentSoftColor, 60))
	}
	ui.DrawRoundedPanel(screen, area, 22, ui.PanelShadowColor, outline, fill)
	ui.DrawText(screen, title, ui.HeadingFace(), area.X+20, area.Y+18, ui.TextLightColor)
	ui.DrawText(screen, detail, ui.SmallFace(), area.X+20, area.Y+52, ui.TextSoftColor)
}

func launchSetupColors() []sim.TeamColor {
	return []sim.TeamColor{sim.TeamColorBlack, sim.TeamColorOrange, sim.TeamColorGreen, sim.TeamColorBlue, sim.TeamColorRed}
}

func launchSetupOpponentColor(home sim.TeamColor) sim.TeamColor {
	colors := launchSetupColors()
	for index, candidate := range colors {
		if candidate == home {
			return colors[(index+1)%len(colors)]
		}
	}
	return sim.TeamColorRed
}

func drawColorChip(screen *ebiten.Image, area ui.Rect, teamColor sim.TeamColor, selected, hovered bool) {
	palette := paletteForTeamColor(teamColor)
	fill := ui.PanelInsetColor
	outline := ui.PanelStrokeColor
	textColor := ui.TextSoftColor
	if hovered {
		outline = ui.PanelStrokeBrightColor
	}
	if selected {
		fill = ui.PanelAltColor
		outline = ui.AccentColor
		textColor = ui.TextLightColor
		ui.DrawGlow(screen, area, 18, ui.WithAlpha(ui.AccentSoftColor, 74))
	}
	ui.DrawRoundedPanel(screen, area, 18, ui.PanelShadowColor, outline, fill)
	vector.FillCircle(screen, float32(area.X+22), float32(area.Y+area.H/2), 9, palette.Primary, true)
	vector.StrokeCircle(screen, float32(area.X+22), float32(area.Y+area.H/2), 9, 2, palette.Trim, true)
	ui.DrawText(screen, strings.ToUpper(TeamColorLabel(teamColor)), ui.SmallFace(), area.X+40, area.Y+10, textColor)
}

func drawRoomCard(screen *ebiten.Image, area ui.Rect, room discovery.Room, selected, hovered bool) {
	fill := ui.PanelInsetColor
	outline := ui.PanelStrokeColor
	if hovered {
		outline = ui.PanelStrokeBrightColor
	}
	if selected {
		fill = ui.PanelAltColor
		outline = ui.AccentColor
		ui.DrawGlow(screen, area, 18, ui.WithAlpha(ui.AccentSoftColor, 66))
	}
	if !room.Joinable() {
		outline = color.RGBA{0x6f, 0x77, 0x84, 0xff}
	}
	ui.DrawRoundedPanel(screen, area, 22, ui.PanelShadowColor, outline, fill)
	palette := paletteForTeamColor(sim.TeamColorBlue)
	vector.FillCircle(screen, float32(area.X+34), float32(area.Y+area.H/2), 16, palette.Primary, true)
	vector.StrokeCircle(screen, float32(area.X+34), float32(area.Y+area.H/2), 16, 3, palette.Trim, true)

	ui.DrawText(screen, strings.ToUpper(room.Code), ui.BodyFace(), area.X+66, area.Y+18, ui.TextLightColor)
	ui.DrawText(screen, truncateLabel(room.Name, 30), ui.BodyFace(), area.X+134, area.Y+18, ui.TextLightColor)
	ui.DrawText(screen, room.Addr, ui.SmallFace(), area.X+134, area.Y+48, ui.TextSoftColor)

	badge := ui.Rect{X: area.X + area.W - 156, Y: area.Y + 18, W: 126, H: 36}
	badgeFill := ui.PanelColor
	badgeOutline := ui.PanelStrokeColor
	badgeLabel := fmt.Sprintf("%d/%d players", room.Status.Players, room.Status.Capacity)
	badgeText := ui.TextSoftColor
	if !room.Joinable() {
		badgeFill = color.RGBA{0x2b, 0x22, 0x26, 0xff}
		badgeOutline = ui.DangerColor
		badgeLabel = "ROOM FULL"
		badgeText = ui.TextLightColor
	}
	ui.DrawRoundedPanel(screen, badge, 16, color.RGBA{0, 0, 0, 0}, badgeOutline, badgeFill)
	ui.DrawTextCentered(screen, badgeLabel, ui.SmallFace(), badge.X+badge.W/2, badge.Y+9, badgeText)
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
