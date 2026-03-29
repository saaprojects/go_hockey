package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	clientapp "hockeyv2/internal/client/app"
	"hockeyv2/internal/server"
	"hockeyv2/internal/sim"
)

var (
	runLauncher  = clientapp.RunLauncher
	runHosted    = clientapp.RunHosted
	runRemote    = clientapp.RunRemote
	runDedicated = server.RunDedicated
	smokeSummary = sim.SmokeSummary
)

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("go-hockey", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	headless := flags.Bool("headless", false, "run a one-tick sim smoke test instead of the playable client")
	serverOnly := flags.Bool("server", false, "run the dedicated multiplayer server only")
	host := flags.Bool("host", false, "start a local multiplayer server and join it with the online client")
	listenAddr := flags.String("listen", ":4242", "TCP listen address for the multiplayer server")
	joinAddr := flags.String("join", "", "join a multiplayer server at host:port")
	if err := flags.Parse(args); err != nil {
		return err
	}

	switch {
	case *headless:
		_, err := fmt.Fprint(stdout, smokeSummary())
		return err
	case *serverOnly:
		return runDedicated(*listenAddr)
	case *host:
		return runHosted(*listenAddr)
	case *joinAddr != "":
		return runRemote(*joinAddr)
	default:
		return runLauncher()
	}
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}
