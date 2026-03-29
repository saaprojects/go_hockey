package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"hockeyv2/internal/client"
	"hockeyv2/internal/discovery"
	"hockeyv2/internal/server"
	"hockeyv2/internal/sim"
)

func main() {
	headless := flag.Bool("headless", false, "run a one-tick sim smoke test instead of the playable client")
	serverOnly := flag.Bool("server", false, "run the dedicated multiplayer server only")
	host := flag.Bool("host", false, "start a local multiplayer server and join it with the online client")
	listenAddr := flag.String("listen", ":4242", "TCP listen address for the multiplayer server")
	joinAddr := flag.String("join", "", "join a multiplayer server at host:port")
	flag.Parse()

	if *headless {
		state := sim.NewGameState()
		sim.Step(&state, nil)
		fmt.Printf(
			"Go Hockey ready. tick=%d home=%d away=%d faceoff=%d puck=(%.0f, %.0f)\n",
			state.Tick,
			len(state.HomeSkaters),
			len(state.AwaySkaters),
			state.FaceoffTicks,
			state.Puck.Position.X,
			state.Puck.Position.Y,
		)
		return
	}

	if *serverOnly {
		runServer(*listenAddr)
		return
	}

	if *host {
		runHost(*listenAddr)
		return
	}

	if *joinAddr != "" {
		if err := client.RunRemote(*joinAddr); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := client.RunApp(); err != nil {
		log.Fatal(err)
	}
}

func runServer(listenAddr string) {
	srv, err := server.Listen(listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	advertiser, err := discovery.NewAdvertiser(srv.Addr(), func() discovery.Status {
		return discovery.Status{Players: srv.PlayerCount(), Capacity: 2}
	})
	if err != nil {
		log.Printf("LAN discovery unavailable: %v", err)
	} else {
		defer advertiser.Close()
		room := advertiser.Room()
		log.Printf("LAN room %s (%s) is discoverable on the local network", room.Code, room.Name)
	}

	log.Printf("Go Hockey server listening on %s", srv.Addr())
	if err := srv.Serve(); err != nil {
		log.Fatal(err)
	}
}

func runHost(listenAddr string) {
	srv, err := server.Listen(listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve()
	}()

	advertiser, err := discovery.NewAdvertiser(srv.Addr(), func() discovery.Status {
		return discovery.Status{Players: srv.PlayerCount(), Capacity: 2}
	})
	if err != nil {
		log.Printf("LAN discovery unavailable: %v", err)
	} else {
		defer advertiser.Close()
		room := advertiser.Room()
		log.Printf("LAN room %s (%s) is discoverable on the local network", room.Code, room.Name)
	}

	joinAddr := localJoinAddress(srv.Addr())
	log.Printf("Hosting Go Hockey on %s", srv.Addr())
	log.Printf("Local client joining %s", joinAddr)
	time.Sleep(150 * time.Millisecond)

	clientErr := client.RunRemote(joinAddr)
	_ = srv.Close()
	select {
	case err := <-serveErr:
		if err != nil {
			log.Printf("server stopped: %v", err)
		}
	default:
	}
	if clientErr != nil {
		log.Fatal(clientErr)
	}
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
