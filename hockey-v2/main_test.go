package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunHeadlessWritesSmokeSummary(t *testing.T) {
	oldSmokeSummary := smokeSummary
	t.Cleanup(func() {
		smokeSummary = oldSmokeSummary
	})
	smokeSummary = func() string { return "ready\n" }

	var output bytes.Buffer
	if err := run([]string{"-headless"}, &output); err != nil {
		t.Fatalf("run headless: %v", err)
	}
	if got := output.String(); got != "ready\n" {
		t.Fatalf("expected smoke output, got %q", got)
	}
}

func TestRunDispatchesDedicatedServer(t *testing.T) {
	oldRunDedicated := runDedicated
	t.Cleanup(func() {
		runDedicated = oldRunDedicated
	})

	called := ""
	runDedicated = func(addr string) error {
		called = addr
		return nil
	}

	if err := run([]string{"-server", "-listen", ":5000"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("run server: %v", err)
	}
	if called != ":5000" {
		t.Fatalf("expected dedicated server listen addr :5000, got %q", called)
	}
}

func TestRunDispatchesHostedClient(t *testing.T) {
	oldRunHosted := runHosted
	t.Cleanup(func() {
		runHosted = oldRunHosted
	})

	called := ""
	runHosted = func(addr string) error {
		called = addr
		return nil
	}

	if err := run([]string{"-host", "-listen", ":5001"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("run host: %v", err)
	}
	if called != ":5001" {
		t.Fatalf("expected hosted client listen addr :5001, got %q", called)
	}
}

func TestRunDispatchesRemoteClient(t *testing.T) {
	oldRunRemote := runRemote
	t.Cleanup(func() {
		runRemote = oldRunRemote
	})

	called := ""
	runRemote = func(addr string) error {
		called = addr
		return nil
	}

	if err := run([]string{"-join", "192.168.1.4:4242"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("run remote: %v", err)
	}
	if called != "192.168.1.4:4242" {
		t.Fatalf("expected remote addr 192.168.1.4:4242, got %q", called)
	}
}

func TestRunDispatchesLauncherByDefault(t *testing.T) {
	oldRunLauncher := runLauncher
	t.Cleanup(func() {
		runLauncher = oldRunLauncher
	})

	called := false
	runLauncher = func() error {
		called = true
		return nil
	}

	if err := run(nil, &bytes.Buffer{}); err != nil {
		t.Fatalf("run launcher: %v", err)
	}
	if !called {
		t.Fatalf("expected launcher path to be called")
	}
}

func TestRunReturnsErrors(t *testing.T) {
	t.Run("parse error", func(t *testing.T) {
		if err := run([]string{"-not-a-flag"}, &bytes.Buffer{}); err == nil {
			t.Fatalf("expected parse error")
		}
	})

	t.Run("dispatch error", func(t *testing.T) {
		oldRunLauncher := runLauncher
		t.Cleanup(func() {
			runLauncher = oldRunLauncher
		})

		want := errors.New("boom")
		runLauncher = func() error { return want }
		if err := run(nil, &bytes.Buffer{}); !errors.Is(err, want) {
			t.Fatalf("expected %v, got %v", want, err)
		}
	})
}
