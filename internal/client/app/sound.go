package app

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"sync"
	"time"

	"hockeyv2/internal/client/assets"
	"hockeyv2/internal/sim"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

type soundEffect int

const (
	soundEffectFaceoff soundEffect = iota
)

const (
	soundSampleRate = 44100
	goalSoundVolume = 0.205
	faceoffVolume   = 0.82
)

type soundboard struct {
	contextOnce  sync.Once
	context      *audio.Context
	clips        map[soundEffect][]byte
	goalClips    map[sim.TeamColor][]byte
	fallbackGoal []byte
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
	return &soundboard{
		clips: map[soundEffect][]byte{
			soundEffectFaceoff: tonePCM(560, 430, 85*time.Millisecond, 0.18),
		},
		goalClips:    loadGoalSoundClips(fallbackGoal),
		fallbackGoal: fallbackGoal,
	}
}

func loadGoalSoundClips(fallback []byte) map[sim.TeamColor][]byte {
	clips := map[sim.TeamColor][]byte{}
	assetPaths := map[sim.TeamColor]string{
		sim.TeamColorBlack:  "sounds/LA - Black.wav",
		sim.TeamColorOrange: "sounds/Anaheim - Orange.wav",
		sim.TeamColorGreen:  "sounds/Vancouver - Green.wav",
		sim.TeamColorBlue:   "sounds/NYR - Blue.wav",
		sim.TeamColorRed:    "sounds/Carolina - RED.wav",
	}
	for teamColor, path := range assetPaths {
		clip := decodeWAVAsset(path)
		if len(clip) == 0 {
			clip = fallback
		}
		clips[teamColor] = clip
	}
	return clips
}

func decodeWAVAsset(path string) []byte {
	data, err := assets.SoundFiles.ReadFile(path)
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

func (s *soundboard) ensureContext() {
	if s == nil {
		return
	}
	s.contextOnce.Do(func() {
		s.context = audio.NewContext(soundSampleRate)
	})
}

func (s *soundboard) Play(effect soundEffect) {
	if s == nil {
		return
	}
	s.playClip(s.clips[effect], faceoffVolume)
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
		return
	}
	if previous.FaceoffTicks > 0 && current.FaceoffTicks == 0 {
		s.Play(soundEffectFaceoff)
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

func tonePCM(startFreq, endFreq float64, duration time.Duration, volume float64) []byte {
	sampleCount := int(float64(soundSampleRate) * duration.Seconds())
	if sampleCount <= 0 {
		return nil
	}
	pcm := make([]byte, sampleCount*4)
	phase := 0.0
	for index := 0; index < sampleCount; index++ {
		progress := float64(index) / float64(sampleCount)
		freq := startFreq + (endFreq-startFreq)*progress
		envelope := 1.0
		attackSamples := int(0.08 * float64(sampleCount))
		releaseSamples := int(0.18 * float64(sampleCount))
		if attackSamples > 0 && index < attackSamples {
			envelope = float64(index) / float64(attackSamples)
		}
		if releaseSamples > 0 && index >= sampleCount-releaseSamples {
			releaseProgress := float64(sampleCount-index) / float64(releaseSamples)
			if releaseProgress < envelope {
				envelope = releaseProgress
			}
		}
		sample := math.Sin(phase) * volume * envelope
		writeStereoPCM(pcm, index, sample, sample)
		phase += 2 * math.Pi * freq / soundSampleRate
	}
	return pcm
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

func concatPCM(clips ...[]byte) []byte {
	total := 0
	for _, clip := range clips {
		total += len(clip)
	}
	joined := make([]byte, 0, total)
	for _, clip := range clips {
		joined = append(joined, clip...)
	}
	return joined
}
