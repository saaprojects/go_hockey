package server

import (
	"log"

	"hockeyv2/internal/discovery"
)

func RunDedicated(listenAddr string) error {
	srv, err := Listen(listenAddr)
	if err != nil {
		return err
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
	return srv.Serve()
}
