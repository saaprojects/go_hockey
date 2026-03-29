package server

import (
	"testing"
	"time"

	"hockeyv2/internal/netcode"
	"hockeyv2/internal/sim"
)

func TestFirstTwoClientsGetHomeAndAway(t *testing.T) {
	srv, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve()
	}()
	defer func() {
		_ = srv.Close()
		select {
		case <-serveDone:
		case <-time.After(500 * time.Millisecond):
		}
	}()

	time.Sleep(40 * time.Millisecond)

	clientA, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial first client: %v", err)
	}
	defer clientA.Close()

	clientB, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial second client: %v", err)
	}
	defer clientB.Close()

	if clientA.Team() != sim.TeamHome {
		t.Fatalf("expected first client to be home, got %q", clientA.Team())
	}
	if clientB.Team() != sim.TeamAway {
		t.Fatalf("expected second client to be away, got %q", clientB.Team())
	}
}

func TestThirdClientIsRejectedWhenMatchIsFull(t *testing.T) {
	srv, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve()
	}()
	defer func() {
		_ = srv.Close()
		select {
		case <-serveDone:
		case <-time.After(500 * time.Millisecond):
		}
	}()

	time.Sleep(40 * time.Millisecond)

	clientA, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial first client: %v", err)
	}
	defer clientA.Close()

	clientB, err := netcode.Dial(srv.Addr())
	if err != nil {
		t.Fatalf("dial second client: %v", err)
	}
	defer clientB.Close()

	clientC, err := netcode.Dial(srv.Addr())
	if err == nil {
		clientC.Close()
		t.Fatalf("expected third client to be rejected")
	}
}
