package input

import (
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
)

func MovementVector() sim.Vec2 {
	return movementVectorFromKeys(
		ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft),
		ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight),
		ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp),
		ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown),
	)
}

func movementVectorFromKeys(left, right, up, down bool) sim.Vec2 {
	move := sim.Vec2{}
	if left {
		move.X -= 1
	}
	if right {
		move.X += 1
	}
	if up {
		move.Y -= 1
	}
	if down {
		move.Y += 1
	}
	return move.Normalized()
}
