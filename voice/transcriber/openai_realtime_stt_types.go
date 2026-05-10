package transcriber

import "fmt"

type sessionUpdateMessage struct {
	Type    string               `json:"type"`
	Session sessionUpdateSession `json:"session"`
}

type sessionUpdateSession struct {
	Type  string             `json:"type"`
	Audio sessionUpdateAudio `json:"audio"`
}

type sessionUpdateAudio struct {
	Input sessionUpdateInput `json:"input"`
}

type sessionUpdateInput struct {
	Transcription  sessionTranscription   `json:"transcription"`
	TurnDetection  *sessionTurnDetection  `json:"turn_detection"`
	NoiseReduction *sessionNoiseReduction `json:"noise_reduction,omitempty"`
}

type sessionTranscription struct {
	Model    string `json:"model"`
	Language string `json:"language,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
}

type sessionTurnDetection struct {
	Type              string  `json:"type"`
	CreateResponse    bool    `json:"create_response"`
	Threshold         float64 `json:"threshold,omitempty"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"`
}

type sessionNoiseReduction struct {
	Type string `json:"type"`
}

type audioBufferAppendMessage struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}

type serverEvent struct {
	Type         string `json:"type"`
	Delta        string `json:"delta"`
	Transcript   string `json:"transcript"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Error        *struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type serverError struct {
	ErrorType string
	Code      string
	Message   string
}

func (e *serverError) Error() string {
	return fmt.Sprintf("server error: type=%s code=%s message=%s", e.ErrorType, e.Code, e.Message)
}
