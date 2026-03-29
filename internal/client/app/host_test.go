package app

import "testing"

func TestLocalJoinAddressNormalizesWildcardHosts(t *testing.T) {
	cases := map[string]string{
		":4242":       "127.0.0.1:4242",
		"0.0.0.0:4242": "127.0.0.1:4242",
		"[::]:4242":    "127.0.0.1:4242",
		"192.168.1.4:4242": "192.168.1.4:4242",
	}
	for input, want := range cases {
		if got := localJoinAddress(input); got != want {
			t.Fatalf("localJoinAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoteWindowTitleUppercasesTeam(t *testing.T) {
	if got := remoteWindowTitle("home"); got != "Go Hockey - Online HOME" {
		t.Fatalf("unexpected title %q", got)
	}
}
