package app

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/fs"
	"math"
	"math/rand"
	"path"
	"sort"
	"sync"
	"time"

	"hockeyv2/internal/client/assets"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	soundSampleRate    = 44100
	goalSoundVolume    = 0.205
	arenaAmbientVolume = 0.14
	menuMusicVolume    = 0.04
)

type soundboard struct {
	contextOnce       sync.Once
	context           *audio.Context
	goalClips         map[sim.TeamColor][]byte
	ambientClip       []byte
	ambientPlayer     *audio.Player
	menuMusicPaths    []string
	menuMusicPlayer   *audio.Player
	lastMenuMusicPath string
	random            *rand.Rand
	fallbackGoal      []byte
}

var (
	sharedSoundboard     *soundboard
	sharedSoundboardOnce sync.Once
)

func defaultSoundboard() *soundboard {
	sharedSoundboardOnce.Do(func() {
		sharedSoundboard = newSoundboard()
	})
	return sharedSoundboard
}

func newSoundboard() *soundboard {
	fallbackGoal := goalHornPCM()
	seed := time.Now().UnixNano()
	return &soundboard{
		goalClips:      loadGoalSoundClips(fallbackGoal),
		ambientClip:    decodeFirstAvailableMP3Asset(arenaAmbientAssetPaths()),
		menuMusicPaths: listMenuMusicAssetPaths(),
		random:         rand.New(rand.NewSource(seed)),
		fallbackGoal:   fallbackGoal,
	}
}

func loadGoalSoundClips(fallback []byte) map[sim.TeamColor][]byte {
	clips := map[sim.TeamColor][]byte{}
	for _, teamColor := range []sim.TeamColor{sim.TeamColorBlack, sim.TeamColorOrange, sim.TeamColorGreen, sim.TeamColorBlue, sim.TeamColorRed} {
		clip := decodeFirstAvailableWAVAsset(goalSoundAssetPaths(teamColor))
		if len(clip) == 0 {
			clip = fallback
		}
		clips[teamColor] = clip
	}
	return clips
}

func goalSoundAssetPaths(teamColor sim.TeamColor) []string {
	switch teamColor {
	case sim.TeamColorBlack:
		return []string{"sounds/goal_black.wav", "sounds/LA - Black.wav"}
	case sim.TeamColorOrange:
		return []string{"sounds/goal_orange.wav", "sounds/Anaheim - Orange.wav"}
	case sim.TeamColorGreen:
		return []string{"sounds/goal_green.wav", "sounds/Vancouver - Green.wav"}
	case sim.TeamColorBlue:
		return []string{"sounds/goal_blue.wav", "sounds/NYR - Blue.wav"}
	case sim.TeamColorRed:
		return []string{"sounds/goal_red.wav", "sounds/Carolina - RED.wav"}
	default:
		return nil
	}
}

func arenaAmbientAssetPaths() []string {
	return []string{"sounds/arena_ambient.mp3", "sounds/Ambient Hockey Arena.mp3"}
}

func listMenuMusicAssetPaths() []string {
	paths := make([]string, 0, 16)
	err := fs.WalkDir(assets.MusicFiles, "music", func(assetPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() || !supportedMusicAssetPath(assetPath) {
			return nil
		}
		paths = append(paths, assetPath)
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Strings(paths)
	return paths
}

func supportedMusicAssetPath(assetPath string) bool {
	switch path.Ext(assetPath) {
	case ".mp3", ".wav":
		return true
	default:
		return false
	}
}

func chooseRandomMenuMusicPath(paths []string, random *rand.Rand, last string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return paths[0]
	}
	if random == nil {
		random = rand.New(rand.NewSource(1))
	}
	index := random.Intn(len(paths))
	selected := paths[index]
	if selected != last {
		return selected
	}
	for attempts := 0; attempts < len(paths); attempts++ {
		index = random.Intn(len(paths))
		selected = paths[index]
		if selected != last {
			return selected
		}
	}
	return paths[(index+1)%len(paths)]
}

func decodeFirstAvailableWAVAsset(paths []string) []byte {
	for _, assetPath := range paths {
		clip := decodeWAVAsset(assets.SoundFiles, assetPath)
		if len(clip) > 0 {
			return clip
		}
	}
	return nil
}

func decodeFirstAvailableMP3Asset(paths []string) []byte {
	for _, assetPath := range paths {
		clip := decodeMP3Asset(assets.SoundFiles, assetPath)
		if len(clip) > 0 {
			return clip
		}
	}
	return nil
}

func decodeAudioAsset(fileSystem fs.FS, assetPath string) []byte {
	switch path.Ext(assetPath) {
	case ".wav":
		return decodeWAVAsset(fileSystem, assetPath)
	case ".mp3":
		return decodeMP3Asset(fileSystem, assetPath)
	default:
		return nil
	}
}

func decodeWAVAsset(fileSystem fs.FS, assetPath string) []byte {
	data, err := fs.ReadFile(fileSystem, assetPath)
	if err != nil {
		return nil
	}
	stream, err := wav.DecodeWithSampleRate(soundSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil
	}
	clip, err := io.ReadAll(stream)
	if err != nil {
		return nil
	}
	return clip
}

func decodeMP3Asset(fileSystem fs.FS, assetPath string) []byte {
	data, err := fs.ReadFile(fileSystem, assetPath)
	if err != nil {
		return nil
	}
	stream, err := mp3.DecodeWithSampleRate(soundSampleRate, bytes.NewReader(data))
	if err != nil {
		return nil
	}
	clip, err := io.ReadAll(stream)
	if err != nil {
		return nil
	}
	return clip
}

func (s *soundboard) ensureContext() {
	if s == nil {
		return
	}
	s.contextOnce.Do(func() {
		s.context = audio.NewContext(soundSampleRate)
	})
}

func (s *soundboard) PlayGoal(teamColor sim.TeamColor) {
	if s == nil {
		return
	}
	clip := s.goalClips[teamColor]
	if len(clip) == 0 {
		clip = s.fallbackGoal
	}
	s.playClip(clip, goalSoundVolume)
}

func (s *soundboard) PlayArenaAmbience() {
	if s == nil || len(s.ambientClip) == 0 || s.ambientPlayer != nil {
		return
	}
	s.ensureContext()
	if s.context == nil {
		return
	}
	loop := audio.NewInfiniteLoop(bytes.NewReader(s.ambientClip), int64(len(s.ambientClip)))
	player, err := s.context.NewPlayer(loop)
	if err != nil {
		return
	}
	player.SetVolume(arenaAmbientVolume)
	player.Play()
	s.ambientPlayer = player
}

func (s *soundboard) StopArenaAmbience() {
	if s == nil || s.ambientPlayer == nil {
		return
	}
	_ = s.ambientPlayer.Close()
	s.ambientPlayer = nil
}

func (s *soundboard) PlayMenuMusic() {
	if s == nil || len(s.menuMusicPaths) == 0 {
		return
	}
	s.ensureContext()
	if s.context == nil {
		return
	}
	if s.menuMusicPlayer != nil {
		if s.menuMusicPlayer.IsPlaying() {
			return
		}
		_ = s.menuMusicPlayer.Close()
		s.menuMusicPlayer = nil
	}
	for attempts := 0; attempts < len(s.menuMusicPaths); attempts++ {
		assetPath := chooseRandomMenuMusicPath(s.menuMusicPaths, s.random, s.lastMenuMusicPath)
		if assetPath == "" {
			return
		}
		clip := decodeAudioAsset(assets.MusicFiles, assetPath)
		s.lastMenuMusicPath = assetPath
		if len(clip) == 0 {
			continue
		}
		player := s.context.NewPlayerFromBytes(clip)
		player.SetVolume(menuMusicVolume)
		player.Play()
		s.menuMusicPlayer = player
		return
	}
}

func (s *soundboard) StopMenuMusic() {
	if s == nil || s.menuMusicPlayer == nil {
		return
	}
	_ = s.menuMusicPlayer.Close()
	s.menuMusicPlayer = nil
}

func (s *soundboard) playClip(clip []byte, volume float64) {
	if len(clip) == 0 {
		return
	}
	s.ensureContext()
	if s.context == nil {
		return
	}
	player := s.context.NewPlayerFromBytes(clip)
	player.SetVolume(volume)
	player.Play()
}

func playMatchStateSounds(s *soundboard, previous, current sim.GameState) {
	if s == nil {
		return
	}
	if current.Score.Home > previous.Score.Home {
		s.PlayGoal(current.HomeColor)
		return
	}
	if current.Score.Away > previous.Score.Away {
		s.PlayGoal(current.AwayColor)
	}
}

func goalHornPCM() []byte {
	duration := 1350 * time.Millisecond
	sampleCount := int(float64(soundSampleRate) * duration.Seconds())
	if sampleCount <= 0 {
		return nil
	}
	pcm := make([]byte, sampleCount*4)
	baseFreq := 155.0
	fifthFreq := 233.0
	harmonicFreq := 311.0
	for index := 0; index < sampleCount; index++ {
		t := float64(index) / soundSampleRate
		progress := float64(index) / float64(sampleCount)
		envelope := 1.0
		attackSamples := int(0.05 * float64(sampleCount))
		releaseSamples := int(0.28 * float64(sampleCount))
		if attackSamples > 0 && index < attackSamples {
			envelope = float64(index) / float64(attackSamples)
		}
		if releaseSamples > 0 && index >= sampleCount-releaseSamples {
			releaseProgress := float64(sampleCount-index) / float64(releaseSamples)
			if releaseProgress < envelope {
				envelope = releaseProgress
			}
		}
		if progress > 0.42 {
			envelope *= 1.0 - (progress-0.42)*0.18
		}
		left := 0.52*hornVoice(t, baseFreq-1.3, 0.0) + 0.26*hornVoice(t, fifthFreq-0.7, 0.35) + 0.12*hornVoice(t, harmonicFreq-0.2, 0.7)
		right := 0.52*hornVoice(t, baseFreq+1.3, 0.12) + 0.26*hornVoice(t, fifthFreq+0.7, 0.48) + 0.12*hornVoice(t, harmonicFreq+0.2, 0.82)
		left = softClip(left * envelope * 0.22)
		right = softClip(right * envelope * 0.22)
		writeStereoPCM(pcm, index, left, right)
	}
	return pcm
}

func hornVoice(t, freq, phaseOffset float64) float64 {
	vibrato := 0.16 * math.Sin(2*math.Pi*4.2*t+phaseOffset)
	phase := 2*math.Pi*freq*t + vibrato
	return math.Sin(phase) + 0.48*math.Sin(2*phase+0.08) + 0.2*math.Sin(3*phase+0.16) + 0.08*math.Sin(4*phase+0.22)
}

func writeStereoPCM(pcm []byte, index int, left, right float64) {
	offset := index * 4
	leftValue := int16(clampAudio(left) * math.MaxInt16)
	rightValue := int16(clampAudio(right) * math.MaxInt16)
	binary.LittleEndian.PutUint16(pcm[offset:], uint16(leftValue))
	binary.LittleEndian.PutUint16(pcm[offset+2:], uint16(rightValue))
}

func clampAudio(value float64) float64 {
	if value > 1.0 {
		return 1.0
	}
	if value < -1.0 {
		return -1.0
	}
	return value
}

func softClip(value float64) float64 {
	return value / (1.0 + math.Abs(value)*0.65)
}
