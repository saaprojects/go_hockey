package app

import (
	"errors"
	"net"
	"os"
	"strings"
)

const (
	onlineServerEnvVar     = "GO_HOCKEY_ONLINE_ADDR"
	defaultOnlineServerAddr = "127.0.0.1:4242"
	onlineRoomCodeLength   = 5
	onlineRoomNameMaxRunes = 28
)

func onlineServerAddress() string {
	if override := strings.TrimSpace(os.Getenv(onlineServerEnvVar)); override != "" {
		return override
	}
	return defaultOnlineServerAddr
}

func defaultOnlineRoomName() string {
	hostName, err := os.Hostname()
	hostName = strings.TrimSpace(hostName)
	if err != nil || hostName == "" {
		return "Online Room"
	}
	return hostName + "'s Room"
}

func normalizedOnlineRoomName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultOnlineRoomName()
	}
	return trimmed
}

func onlineConnectionErrorStatus(err error) string {
	if err == nil {
		return ""
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return "Unable to reach the room server"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "Unable to reach the room server"
	}
	return message
}
