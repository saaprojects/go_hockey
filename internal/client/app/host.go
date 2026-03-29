package app

import (
	"fmt"
	"strings"
	"time"

	"hockeyv2/internal/discovery"
	"hockeyv2/internal/netcode"
	"hockeyv2/internal/server"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2"
)

func startHostedSession(listenAddr string) (*server.Server, chan error, *discovery.Advertiser, *RemoteGame, error) {
	srv, err := server.Listen(listenAddr)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	advertiser, err := discovery.NewAdvertiser(srv.Addr(), func() discovery.Status {
		return discovery.Status{Players: srv.PlayerCount(), Capacity: 2}
	})
	if err != nil {
		_ = srv.Close()
		return nil, nil, nil, nil, err
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve()
	}()
	joinAddr := localJoinAddress(srv.Addr())
	time.Sleep(150 * time.Millisecond)
	clientConn, err := netcode.Dial(joinAddr)
	if err != nil {
		_ = advertiser.Close()
		_ = srv.Close()
		return nil, nil, nil, nil, err
	}
	game := newRemoteGame(clientConn)
	return srv, serveErr, advertiser, game, nil
}

func localJoinAddress(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	if strings.HasPrefix(addr, "[::]:") {
		return "127.0.0.1:" + strings.TrimPrefix(addr, "[::]:")
	}
	return addr
}

func setWindowDefaults(title string) {
	ebiten.SetWindowSize(int(sim.WindowWidth), int(sim.WindowHeight))
	ebiten.SetWindowTitle(title)
	ebiten.SetTPS(sim.TickRate)
}

func remoteWindowTitle(localTeam string) string {
	return fmt.Sprintf("Go Hockey - Online %s", strings.ToUpper(localTeam))
}
