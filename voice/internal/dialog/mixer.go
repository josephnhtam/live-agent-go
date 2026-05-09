package dialog

import (
	"context"
	"github.com/josephnhtam/live-agent-go/voice/audio"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/josephnhtam/live-agent-go/voice/internal/core"
)

const (
	mixerTickInterval = 20 * time.Millisecond
	int16MaxValue     = 32767
	int16MinValue     = -32768
)

type bgTrack struct {
	wave     *audio.Wave
	opts     *audio.Options
	position int
	volume   float64
	stopped  atomic.Bool
}

func (t *bgTrack) Stop() {
	t.stopped.Store(true)
}

func (t *bgTrack) SetVolume(volume float64) {
	t.opts.WithVolume(volume)
}

type mixerCommand int

const (
	cmdSetSpeechSource mixerCommand = iota
	cmdStopAllTracks
)

type mixerMsg struct {
	cmd mixerCommand
	obj any
}

type mixer struct {
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	out            chan<- core.AudioFrame
	sampleRate     int32
	samplesPerTick int

	speechCh        <-chan core.AudioFrame
	speechSampleBuf []int16
	tracks          []*bgTrack
	trackCh         chan *bgTrack
	cmdCh           chan mixerMsg

	mutex  sync.Mutex
	logger *slog.Logger
}

func newMixer(out chan<- core.AudioFrame, sampleRate int32, logger *slog.Logger) *mixer {
	ctx, cancel := context.WithCancel(context.Background())

	return &mixer{
		ctx:            ctx,
		cancel:         cancel,
		wg:             sync.WaitGroup{},
		out:            out,
		sampleRate:     sampleRate,
		samplesPerTick: int(sampleRate) * int(mixerTickInterval) / int(time.Second),
		trackCh:        make(chan *bgTrack, 16),
		cmdCh:          make(chan mixerMsg, 4),
		logger:         logger,
	}
}

func (m *mixer) Close(ctx context.Context) error {
	m.cancel()

	return helper.WaitWithCtx(ctx, &m.wg)
}

func (m *mixer) SetSpeechSource(ch <-chan core.AudioFrame) {
	select {
	case m.cmdCh <- mixerMsg{cmd: cmdSetSpeechSource, obj: ch}:
	case <-m.ctx.Done():
	}
}

func (m *mixer) AddTrack(wave *audio.Wave, opts *audio.Options) (*bgTrack, error) {
	if wave == nil {
		return nil, ErrNilWave
	}

	if wave.SampleRate() != m.sampleRate {
		return nil, ErrSampleRateMismatch
	}

	if opts == nil {
		opts = audio.NewOptions()
	}

	track := &bgTrack{
		wave:   wave,
		opts:   opts,
		volume: opts.Volume(),
	}

	select {
	case m.trackCh <- track:
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}

	return track, nil
}

func (m *mixer) StopAllTracks() {
	select {
	case m.cmdCh <- mixerMsg{cmd: cmdStopAllTracks}:
	case <-m.ctx.Done():
	}
}

func (m *mixer) Run() {
	defer close(m.out)

	m.wg.Add(1)
	defer m.wg.Done()

	timer := time.NewTimer(mixerTickInterval)
	defer timer.Stop()

	for {
		if m.ctx.Err() != nil {
			return
		}

		m.collectNewTracks()

		if m.speechCh != nil || len(m.speechSampleBuf) > 0 {
			m.runSpeechActive(timer)
		} else if m.hasActiveTracks() {
			m.runBGOnly(timer)
		} else {
			m.runIdle()
		}
	}
}

func (m *mixer) runIdle() {
	select {
	case <-m.ctx.Done():

	case msg := <-m.cmdCh:
		m.handleCmd(msg)

	case track := <-m.trackCh:
		m.mutex.Lock()
		m.tracks = append(m.tracks, track)
		m.mutex.Unlock()
	}
}

func (m *mixer) runSpeechActive(timer *time.Timer) {
	select {
	case <-m.ctx.Done():
		return

	case <-timer.C:
		defer timer.Reset(mixerTickInterval)

		if m.ctx.Err() != nil {
			return
		}

		m.drainSpeechFrames()
		m.drainCommands()
		m.collectNewTracks()

		if len(m.speechSampleBuf) >= m.samplesPerTick {
			samples := m.speechSampleBuf[:m.samplesPerTick]
			m.speechSampleBuf = m.speechSampleBuf[m.samplesPerTick:]
			m.emitSpeechSamples(samples)
		} else if m.speechCh == nil {
			if len(m.speechSampleBuf) > 0 {
				m.emitSpeechSamples(m.speechSampleBuf)
				m.speechSampleBuf = m.speechSampleBuf[:0]
			}
		} else {
			m.emitBGOnlyFrame()
		}
	}
}

func (m *mixer) drainSpeechFrames() {
	for {
		select {
		case frame, ok := <-m.speechCh:
			if !ok {
				m.speechCh = nil
				return
			}

			if pcm, ok := frame.(*core.PCMFrame); ok {
				if pcm.Context() != nil && pcm.Context().Err() != nil {
					continue
				}

				m.speechSampleBuf = append(m.speechSampleBuf, pcm.PCMData...)
			}
		default:
			return
		}
	}
}

func (m *mixer) runBGOnly(timer *time.Timer) {
	select {
	case <-m.ctx.Done():
		return

	case <-timer.C:
		defer timer.Reset(mixerTickInterval)

		if m.ctx.Err() != nil {
			return
		}

		if m.speechCh != nil {
			return
		}

		m.drainCommands()
		m.collectNewTracks()

		m.emitBGOnlyFrame()
	}
}

func (m *mixer) handleCmd(msg mixerMsg) {
	switch msg.cmd {
	case cmdSetSpeechSource:
		m.speechSampleBuf = m.speechSampleBuf[:0]
		if speechCh, ok := msg.obj.(<-chan core.AudioFrame); ok {
			m.speechCh = speechCh
		}

	case cmdStopAllTracks:
		m.mutex.Lock()
		m.tracks = m.tracks[:0]
		m.mutex.Unlock()
	}
}

func (m *mixer) drainCommands() {
	for {
		select {
		case msg := <-m.cmdCh:
			m.handleCmd(msg)
		default:
			return
		}
	}
}

func (m *mixer) emitSpeechSamples(samples []int16) {
	mixed := make([]int32, len(samples))
	for i, s := range samples {
		mixed[i] = int32(s)
	}

	m.mutex.Lock()
	m.mixTracks(mixed, true)
	m.mutex.Unlock()

	m.sendFrame(&core.PCMFrame{
		PCMData:      clampToInt16(mixed),
		SampleRateHz: m.sampleRate,
		NumChannels:  1,
		Ctx:          m.ctx,
	})
}

func (m *mixer) emitBGOnlyFrame() {
	mixed := make([]int32, m.samplesPerTick)

	m.mutex.Lock()
	m.mixTracks(mixed, false)
	m.mutex.Unlock()

	clamped := clampToInt16(mixed)

	m.sendFrame(&core.PCMFrame{
		PCMData:      clamped,
		SampleRateHz: m.sampleRate,
		NumChannels:  1,
		Ctx:          m.ctx,
	})
}

func (m *mixer) collectNewTracks() {
	for {
		select {
		case track := <-m.trackCh:
			m.mutex.Lock()
			m.tracks = append(m.tracks, track)
			m.mutex.Unlock()
		default:
			return
		}
	}
}

func (m *mixer) sendFrame(frame core.AudioFrame) {
	select {
	case m.out <- frame:
	case <-m.ctx.Done():
	}
}

func (m *mixer) hasActiveTracks() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.tracks) > 0
}

func (m *mixer) mixTracks(mixed []int32, speechPlaying bool) {
	length := len(mixed)
	alive := m.tracks[:0]

	for _, track := range m.tracks {
		if track.stopped.Load() {
			continue
		}

		samples := m.readTrackSamples(track, length, speechPlaying)

		if len(samples) == 0 && track.position >= len(track.wave.Samples()) && !track.opts.Loop() {
			continue
		}

		for idx, sample := range samples {
			mixed[idx] += int32(sample)
		}

		if track.position >= len(track.wave.Samples()) && !track.opts.Loop() {
			continue
		}

		alive = append(alive, track)
	}

	for idx := len(alive); idx < len(m.tracks); idx++ {
		m.tracks[idx] = nil
	}

	m.tracks = alive
}

func (m *mixer) readTrackSamples(track *bgTrack, length int, speechPlaying bool) []int16 {
	samples := track.wave.Samples()
	waveLen := len(samples)

	if waveLen == 0 {
		return nil
	}

	out := make([]int16, length)

	for idx := 0; idx < length; idx++ {
		if track.position >= waveLen {
			if track.opts.Loop() {
				track.position = 0
			} else {
				out = out[:idx]
				break
			}
		}

		out[idx] = samples[track.position]
		track.position++
	}

	m.applyVolume(track, out, speechPlaying)
	return out
}

func (m *mixer) applyVolume(track *bgTrack, samples []int16, speechPlaying bool) {
	volume := track.opts.Volume()

	if speechPlaying && track.opts.Duck() {
		volume = track.opts.DuckVolume()
	}

	for idx := range samples {
		samples[idx] = int16(float64(samples[idx]) * volume)
	}
}

func clampToInt16(samples []int32) []int16 {
	out := make([]int16, len(samples))

	for idx, sample := range samples {
		if sample > int16MaxValue {
			sample = int16MaxValue
		} else if sample < int16MinValue {
			sample = int16MinValue
		}

		out[idx] = int16(sample)
	}

	return out
}
