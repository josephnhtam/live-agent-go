package voice

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/josephnhtam/live-agent-go/voice/internal/dialog"
	"github.com/josephnhtam/live-agent-go/voice/internal/speech"
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
	if !r.responder.IsInterruptible() {
		return
	}

	r.stopInterruptTimer()

	if r.minInterruptDuration <= 0 {
		r.responder.CancelResponse(context.Background())
		return
	}

	r.mutex.Lock()
	r.interruptTimer = time.AfterFunc(r.minInterruptDuration, func() {
		r.responder.CancelResponse(context.Background())
	})
	r.mutex.Unlock()
}

func (r *recognitionHandler) OnInterim() {
	if !r.responder.IsInterruptible() {
		return
	}

	if !r.interruptOnInterim {
		return
	}

	r.responder.CancelResponse(context.Background())
}

func (r *recognitionHandler) OnSpeechEnd() {
	r.stopInterruptTimer()
}

func (r *recognitionHandler) OnSpeechRecognized(transcripts []Transcript) {
	if !r.responder.IsInterruptible() {
		return
	}

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

func combineTranscripts(transcripts []Transcript) string {
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
