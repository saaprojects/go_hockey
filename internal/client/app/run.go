package app

import (
	"log"

	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
)

func RunLauncher() error {
	app := NewApp()
	defer app.Close()
	defer ebiten.SetWindowTitle("Go Hockey")
	defer ebiten.SetTPS(sim.TickRate)
	defer ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))

	setWindowDefaults("Go Hockey")
	return ebiten.RunGame(app)
}

func RunSolo() error {
	game := NewSoloGame()
	game.standalone = true
	setWindowDefaults("Go Hockey - Solo")
	return ebiten.RunGame(game)
}

func RunRemote(addr string) error {
	game, err := NewRemoteGame(addr)
	if err != nil {
		return err
	}
	defer game.Close()
	setWindowDefaults(remoteWindowTitle(string(game.localTeam)))
	return ebiten.RunGame(game)
}

func RunHosted(listenAddr string) error {
	srv, serveErr, advertiser, game, err := startHostedSession(listenAddr, sim.TeamColorBlue)
	if err != nil {
		return err
	}
	defer func() {
		_ = game.Close()
		_ = advertiser.Close()
		_ = srv.Close()
		select {
		case err := <-serveErr:
			if err != nil {
				log.Printf("server stopped: %v", err)
			}
		default:
		}
	}()
	game.standalone = true
	setWindowDefaults("Go Hockey - Host Multiplayer")
	return ebiten.RunGame(game)
}
