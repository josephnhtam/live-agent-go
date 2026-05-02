package voice

import (
	"context"
	"live-agent-go/voice/core"
	"live-agent-go/voice/internal/dialog"
	"live-agent-go/voice/internal/speech"
	"strings"
	"sync"
)

type recognitionHandlerConfig struct {
	Ctx       context.Context
	Responder *dialog.Responder
}

type recognitionHandler struct {
	ctx       context.Context
	responder *dialog.Responder

	mutex      sync.Mutex
	cancelResp context.CancelFunc
	respWg     *sync.WaitGroup
}

var _ speech.RecognitionHandler = (*recognitionHandler)(nil)

func newRecognitionHandler(config recognitionHandlerConfig) *recognitionHandler {
	return &recognitionHandler{
		ctx:        config.Ctx,
		responder:  config.Responder,
		mutex:      sync.Mutex{},
		cancelResp: nil,
	}
}

func (r *recognitionHandler) OnSpeechStart() {
	r.CancelResponse()
}

func (r *recognitionHandler) OnSpeechRecognized(transcripts []core.Transcript) {
	prompt := combineTranscripts(transcripts)

	r.CancelResponse()
	
	ctx := r.createResponseContext()
	wg := r.responder.Respond(ctx, prompt)

	r.mutex.Lock()
	r.respWg = wg
	r.mutex.Unlock()
}

func (r *recognitionHandler) CancelResponse() {
	r.mutex.Lock()

	cancel := r.cancelResp
	wg := r.respWg

	r.cancelResp = nil
	r.respWg = nil

	r.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	if wg != nil {
		wg.Wait()
	}
}

func (r *recognitionHandler) createResponseContext() (ctx context.Context) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ctx, r.cancelResp = context.WithCancel(r.ctx)
	return
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
