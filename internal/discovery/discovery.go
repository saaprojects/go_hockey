package discovery

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultUDPPort       = 4242
	defaultProbeInterval = time.Second
	defaultEntryTTL      = 4 * time.Second
	discoveryMagic       = "go-hockey-lan-v1"
	packetKindProbe      = "probe"
	packetKindRoom       = "room"
	roomCodeAlphabet     = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	roomCodeLength       = 4
)

type Status struct {
	Players  int `json:"players"`
	Capacity int `json:"capacity"`
}

type Room struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Addr   string `json:"addr"`
	Status Status `json:"status"`
}

func (r Room) Joinable() bool {
	return r.Status.Capacity == 0 || r.Status.Players < r.Status.Capacity
}

type AdvertiserConfig struct {
	ListenAddr string
	TCPAddr    string
	RoomName   string
	RoomCode   string
	StatusFunc func() Status
}

type BrowserConfig struct {
	ListenAddr    string
	DiscoveryPort int
	ProbeTargets  []string
	ProbeInterval time.Duration
	EntryTTL      time.Duration
}

type Advertiser struct {
	conn       *net.UDPConn
	room       Room
	tcpAddr    string
	statusFunc func() Status
	closed     chan struct{}
	closeOnce  sync.Once
}

type Browser struct {
	conn      *net.UDPConn
	cfg       BrowserConfig
	updates   chan []Room
	rooms     map[string]discoveredRoom
	mu        sync.Mutex
	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

type discoveredRoom struct {
	room     Room
	lastSeen time.Time
}

type packet struct {
	Magic    string `json:"magic"`
	Kind     string `json:"kind"`
	Code     string `json:"code,omitempty"`
	Name     string `json:"name,omitempty"`
	Addr     string `json:"addr,omitempty"`
	Players  int    `json:"players,omitempty"`
	Capacity int    `json:"capacity,omitempty"`
}

func NewAdvertiser(tcpAddr string, statusFunc func() Status) (*Advertiser, error) {
	return NewAdvertiserWithConfig(AdvertiserConfig{
		ListenAddr: fmt.Sprintf(":%d", DefaultUDPPort),
		TCPAddr:    tcpAddr,
		StatusFunc: statusFunc,
	})
}

func NewAdvertiserWithConfig(cfg AdvertiserConfig) (*Advertiser, error) {
	if strings.TrimSpace(cfg.TCPAddr) == "" {
		return nil, fmt.Errorf("tcp address is required")
	}
	listenAddr := strings.TrimSpace(cfg.ListenAddr)
	if listenAddr == "" {
		listenAddr = fmt.Sprintf(":%d", DefaultUDPPort)
	}
	roomCode := strings.TrimSpace(strings.ToUpper(cfg.RoomCode))
	if roomCode == "" {
		generated, err := randomRoomCode(rand.Reader)
		if err != nil {
			return nil, err
		}
		roomCode = generated
	}
	roomName := strings.TrimSpace(cfg.RoomName)
	if roomName == "" {
		roomName = defaultRoomName()
	}
	statusFunc := cfg.StatusFunc
	if statusFunc == nil {
		statusFunc = func() Status {
			return Status{Players: 0, Capacity: 2}
		}
	}
	conn, err := net.ListenUDP("udp4", mustResolveUDPAddr(listenAddr))
	if err != nil {
		return nil, err
	}
	a := &Advertiser{
		conn:       conn,
		room:       Room{Code: roomCode, Name: roomName},
		tcpAddr:    cfg.TCPAddr,
		statusFunc: statusFunc,
		closed:     make(chan struct{}),
	}
	go a.serve()
	return a, nil
}

func (a *Advertiser) Addr() string {
	return a.conn.LocalAddr().String()
}

func (a *Advertiser) Room() Room {
	room := a.room
	room.Status = normalizeStatus(a.statusFunc())
	return room
}

func (a *Advertiser) Close() error {
	var err error
	a.closeOnce.Do(func() {
		close(a.closed)
		err = a.conn.Close()
	})
	return err
}

func (a *Advertiser) serve() {
	buffer := make([]byte, 1024)
	for {
		_ = a.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, remote, err := a.conn.ReadFromUDP(buffer)
		if err != nil {
			if isClosedNetworkError(err) {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-a.closed:
					return
				default:
				}
				continue
			}
			continue
		}
		var probe packet
		if err := json.Unmarshal(buffer[:n], &probe); err != nil {
			continue
		}
		if probe.Magic != discoveryMagic || probe.Kind != packetKindProbe {
			continue
		}
		addr, err := a.advertisedAddr(remote)
		if err != nil {
			continue
		}
		status := normalizeStatus(a.statusFunc())
		response, err := json.Marshal(packet{
			Magic:    discoveryMagic,
			Kind:     packetKindRoom,
			Code:     a.room.Code,
			Name:     a.room.Name,
			Addr:     addr,
			Players:  status.Players,
			Capacity: status.Capacity,
		})
		if err != nil {
			continue
		}
		_, _ = a.conn.WriteToUDP(response, remote)
	}
}

func (a *Advertiser) advertisedAddr(remote *net.UDPAddr) (string, error) {
	host, port, err := net.SplitHostPort(a.tcpAddr)
	if err != nil {
		return "", err
	}
	if isUsableAdvertisedHost(host) {
		return net.JoinHostPort(host, port), nil
	}
	ip, err := localIPv4ForRemote(remote)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(ip, port), nil
}

func NewBrowser() (*Browser, error) {
	return NewBrowserWithConfig(BrowserConfig{
		ListenAddr:    ":0",
		DiscoveryPort: DefaultUDPPort,
		ProbeInterval: defaultProbeInterval,
		EntryTTL:      defaultEntryTTL,
	})
}

func NewBrowserWithConfig(cfg BrowserConfig) (*Browser, error) {
	listenAddr := strings.TrimSpace(cfg.ListenAddr)
	if listenAddr == "" {
		listenAddr = ":0"
	}
	if cfg.DiscoveryPort == 0 {
		cfg.DiscoveryPort = DefaultUDPPort
	}
	if cfg.ProbeInterval <= 0 {
		cfg.ProbeInterval = defaultProbeInterval
	}
	if cfg.EntryTTL <= 0 {
		cfg.EntryTTL = defaultEntryTTL
	}
	conn, err := net.ListenUDP("udp4", mustResolveUDPAddr(listenAddr))
	if err != nil {
		return nil, err
	}
	b := &Browser{
		conn:    conn,
		cfg:     cfg,
		updates: make(chan []Room, 1),
		rooms:   make(map[string]discoveredRoom),
		closed:  make(chan struct{}),
	}
	b.wg.Add(2)
	go b.readLoop()
	go b.probeLoop()
	return b, nil
}

func (b *Browser) Updates() <-chan []Room {
	return b.updates
}

func (b *Browser) Close() error {
	var err error
	b.closeOnce.Do(func() {
		close(b.closed)
		err = b.conn.Close()
		b.wg.Wait()
		close(b.updates)
	})
	return err
}

func (b *Browser) readLoop() {
	defer b.wg.Done()
	buffer := make([]byte, 2048)
	for {
		_ = b.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _, err := b.conn.ReadFromUDP(buffer)
		if err != nil {
			if isClosedNetworkError(err) {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-b.closed:
					return
				default:
				}
				continue
			}
			continue
		}
		var roomPacket packet
		if err := json.Unmarshal(buffer[:n], &roomPacket); err != nil {
			continue
		}
		if roomPacket.Magic != discoveryMagic || roomPacket.Kind != packetKindRoom || strings.TrimSpace(roomPacket.Addr) == "" {
			continue
		}
		b.recordRoom(Room{
			Code: roomPacket.Code,
			Name: roomPacket.Name,
			Addr: roomPacket.Addr,
			Status: normalizeStatus(Status{
				Players:  roomPacket.Players,
				Capacity: roomPacket.Capacity,
			}),
		}, time.Now())
	}
}

func (b *Browser) probeLoop() {
	defer b.wg.Done()
	b.sendProbe()
	b.pruneExpired(time.Now())
	ticker := time.NewTicker(b.cfg.ProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-b.closed:
			return
		case <-ticker.C:
			b.sendProbe()
			b.pruneExpired(time.Now())
		}
	}
}

func (b *Browser) sendProbe() {
	targets := b.probeTargets()
	payload, err := json.Marshal(packet{Magic: discoveryMagic, Kind: packetKindProbe})
	if err != nil {
		return
	}
	for _, target := range targets {
		_, _ = b.conn.WriteToUDP(payload, target)
	}
}

func (b *Browser) probeTargets() []*net.UDPAddr {
	if len(b.cfg.ProbeTargets) > 0 {
		targets := make([]*net.UDPAddr, 0, len(b.cfg.ProbeTargets))
		for _, target := range b.cfg.ProbeTargets {
			addr, err := net.ResolveUDPAddr("udp4", target)
			if err == nil {
				targets = append(targets, addr)
			}
		}
		return targets
	}
	return broadcastTargets(b.cfg.DiscoveryPort)
}

func (b *Browser) recordRoom(room Room, seen time.Time) {
	key := roomKey(room)
	b.mu.Lock()
	defer b.mu.Unlock()
	existing, ok := b.rooms[key]
	if ok && existing.room == room {
		existing.lastSeen = seen
		b.rooms[key] = existing
		return
	}
	b.rooms[key] = discoveredRoom{room: room, lastSeen: seen}
	b.emitSnapshotLocked()
}

func (b *Browser) pruneExpired(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	changed := false
	for key, room := range b.rooms {
		if now.Sub(room.lastSeen) > b.cfg.EntryTTL {
			delete(b.rooms, key)
			changed = true
		}
	}
	if changed {
		b.emitSnapshotLocked()
	}
}

func (b *Browser) emitSnapshotLocked() {
	rooms := make([]Room, 0, len(b.rooms))
	for _, room := range b.rooms {
		rooms = append(rooms, room.room)
	}
	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].Status.Players != rooms[j].Status.Players {
			return rooms[i].Status.Players < rooms[j].Status.Players
		}
		if rooms[i].Name != rooms[j].Name {
			return rooms[i].Name < rooms[j].Name
		}
		if rooms[i].Code != rooms[j].Code {
			return rooms[i].Code < rooms[j].Code
		}
		return rooms[i].Addr < rooms[j].Addr
	})
	select {
	case b.updates <- rooms:
	default:
		select {
		case <-b.updates:
		default:
		}
		select {
		case b.updates <- rooms:
		default:
		}
	}
}

func roomKey(room Room) string {
	return room.Code + "|" + room.Addr
}

func normalizeStatus(status Status) Status {
	if status.Capacity <= 0 {
		status.Capacity = 2
	}
	if status.Players < 0 {
		status.Players = 0
	}
	if status.Players > status.Capacity {
		status.Players = status.Capacity
	}
	return status
}

func isUsableAdvertisedHost(host string) bool {
	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return !strings.EqualFold(host, "localhost")
	}
	return !ip.IsUnspecified() && !ip.IsLoopback()
}

func localIPv4ForRemote(remote *net.UDPAddr) (string, error) {
	conn, err := net.DialUDP("udp4", nil, remote)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || local.IP == nil {
		return "", fmt.Errorf("unable to resolve local address")
	}
	ip := local.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("no ipv4 route available")
	}
	return ip.String(), nil
}

func broadcastTargets(port int) []*net.UDPAddr {
	targets := []*net.UDPAddr{}
	seen := map[string]struct{}{}
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagBroadcast == 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok || ipNet.IP == nil {
					continue
				}
				ip := ipNet.IP.To4()
				if ip == nil || ip.IsLoopback() || len(ipNet.Mask) != 4 {
					continue
				}
				broadcast := make(net.IP, 4)
				for i := range broadcast {
					broadcast[i] = ip[i] | ^ipNet.Mask[i]
				}
				target := net.JoinHostPort(broadcast.String(), strconv.Itoa(port))
				if _, ok := seen[target]; ok {
					continue
				}
				seen[target] = struct{}{}
				targets = append(targets, mustResolveUDPAddr(target))
			}
		}
	}
	limited := net.JoinHostPort("255.255.255.255", strconv.Itoa(port))
	if _, ok := seen[limited]; !ok {
		targets = append(targets, mustResolveUDPAddr(limited))
	}
	return targets
}

func defaultRoomName() string {
	hostName, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostName) == "" {
		return "LAN Host"
	}
	return hostName
}

func randomRoomCode(reader io.Reader) (string, error) {
	buffer := make([]byte, roomCodeLength)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return "", err
	}
	code := make([]byte, roomCodeLength)
	for index, value := range buffer {
		code[index] = roomCodeAlphabet[int(value)%len(roomCodeAlphabet)]
	}
	return string(code), nil
}

func mustResolveUDPAddr(addr string) *net.UDPAddr {
	resolved, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(err)
	}
	return resolved
}

func isClosedNetworkError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "closed network connection")
}
