package voice

import (
	"github.com/josephnhtam/live-agent-go/voice/core"
	"github.com/josephnhtam/live-agent-go/voice/internal/dialog"
	"github.com/josephnhtam/live-agent-go/voice/internal/speech"
	"strings"
	"sync"
	"time"
)

type recognitionHandlerConfig struct {
	Responder            *dialog.Responder
	MinInterruptDuration time.Duration
	InterruptOnInterim   bool
}

type recognitionHandler struct {
	responder *dialog.Responder

	minInterruptDuration time.Duration
	interruptOnInterim   bool

	mutex          sync.Mutex
	interruptTimer *time.Timer
}

var _ speech.RecognitionHandler = (*recognitionHandler)(nil)

func newRecognitionHandler(config recognitionHandlerConfig) *recognitionHandler {
	return &recognitionHandler{
		responder:            config.Responder,
		minInterruptDuration: config.MinInterruptDuration,
		interruptOnInterim:   config.InterruptOnInterim,
	}
}

func (r *recognitionHandler) OnSpeechStart() {
	r.stopInterruptTimer()

	if r.minInterruptDuration <= 0 {
		r.responder.CancelResponse()
		return
	}

	r.mutex.Lock()
	r.interruptTimer = time.AfterFunc(r.minInterruptDuration, r.responder.CancelResponse)
	r.mutex.Unlock()
}

func (r *recognitionHandler) OnInterim() {
	if !r.interruptOnInterim {
		return
	}

	r.responder.CancelResponse()
}

func (r *recognitionHandler) OnSpeechEnd() {
	r.stopInterruptTimer()
}

func (r *recognitionHandler) OnSpeechRecognized(transcripts []core.Transcript) {
	prompt := combineTranscripts(transcripts)

	r.stopInterruptTimer()
	r.responder.Respond(prompt)
}

func (r *recognitionHandler) stopInterruptTimer() {
	r.mutex.Lock()
	t := r.interruptTimer
	r.interruptTimer = nil
	r.mutex.Unlock()

	if t != nil {
		t.Stop()
	}
}

func combineTranscripts(transcripts []core.Transcript) string {
	if len(transcripts) == 0 {
		return ""
	}

	sb := strings.Builder{}
	for _, t := range transcripts {
		sb.WriteString(t.Text)
		sb.WriteRune(' ')
	}

	result := sb.String()
	return result[:len(result)-1]
}
