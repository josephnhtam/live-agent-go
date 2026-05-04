package voice

import (
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/sync/errgroup"
	"live-agent-go/voice/helper"
	"live-agent-go/voice/transport"
	"live-agent-go/voice/transport/types"
	"log/slog"
)

type MessageRole string

const (
	MessageRoleUser  MessageRole = "user"
	MessageRoleAgent MessageRole = "agent"
)

type Message struct {
	MessageID string      `json:"message_id"`
	Role      MessageRole `json:"role"`
	Text      string      `json:"text"`
}

type MessageSerializer interface {
	Serialize(msg Message) (string, error)
}

type DefaultMessageSerializer struct{}

func (d DefaultMessageSerializer) Serialize(msg Message) (string, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type SessionAgentConfig struct {
	AgentConfig  AgentConfig
	AgentOptions []AgentOption
	Logger       *slog.Logger
}

type SessionAgent struct {
	session  transport.Session
	agent    *Agent
	options  *sessionAgentOptions
	audioCh  <-chan AudioFrame
	tokenCh  <-chan Token
	promptCh <-chan Prompt
	logger   *slog.Logger
}

func NewSessionAgent(session types.Session, config SessionAgentConfig, opts ...SessionAgentOption) (*SessionAgent, error) {
	options := buildSessionAgentOptions(opts...)

	audioCh := make(chan AudioFrame, 128)

	agentOpts := []AgentOption{
		SubscribeAudio(audioCh),
	}

	agentOpts = append(agentOpts, config.AgentOptions...)

	tokenCh := make(chan Token, 32)
	promptCh := make(chan Prompt, 32)

	agentOpts = append(agentOpts,
		SubscribeToken(tokenCh),
		SubscribePrompt(promptCh),
	)

	agent, err := NewAgent(config.AgentConfig, agentOpts...)
	if err != nil {
		return nil, err
	}

	logger := config.Logger
	if logger == nil {
		logger = helper.NoopLogger()
	}

	return &SessionAgent{
		session:  session,
		agent:    agent,
		options:  options,
		audioCh:  audioCh,
		tokenCh:  tokenCh,
		promptCh: promptCh,
		logger:   logger.WithGroup("session_agent"),
	}, nil
}

func (a *SessionAgent) Run(ctx context.Context) error {
	if err := a.agent.Start(ctx); err != nil {
		return err
	}

	defer a.agent.Stop(ctx)

	grp, grpCtx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		select {
		case <-a.session.Done():
			return ErrSessionDone
		case <-grpCtx.Done():
			return grpCtx.Err()
		}
	})

	grp.Go(func() error {
		select {
		case err := <-a.agent.Done():
			return err
		case <-grpCtx.Done():
			return grpCtx.Err()
		}
	})

	grp.Go(func() error {
		return a.inboundAudioLoop(grpCtx)
	})

	grp.Go(func() error {
		return a.outboundAudioLoop(grpCtx)
	})

	grp.Go(func() error {
		return a.outboundTokenLoop(grpCtx)
	})

	grp.Go(func() error {
		return a.outboundPromptLoop(grpCtx)
	})

	if a.options.iceBreaking {
		a.agent.IceBreaking()
	}

	return grp.Wait()
}

func (a *SessionAgent) Close(ctx context.Context) error {
	agentErr := a.agent.Stop(ctx)
	sessionErr := a.session.Close()
	return errors.Join(agentErr, sessionErr)
}

func (a *SessionAgent) inboundAudioLoop(ctx context.Context) error {
	for {
		select {
		case frame, ok := <-a.session.AudioIn():
			if !ok {
				return nil
			}

			if err := a.agent.Feed(ctx, frame); err != nil {
				if errors.Is(err, ErrAlreadyStopped) || errors.Is(err, ErrNotStarted) {
					return err
				}

				a.logger.Warn("error feeding audio to agent", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *SessionAgent) outboundAudioLoop(ctx context.Context) error {
	for {
		select {
		case frame, ok := <-a.audioCh:
			if !ok {
				return nil
			}

			if frame.Context() != nil && frame.Context().Err() != nil {
				continue
			}

			if err := a.session.SendAudio(frame); err != nil {
				if errors.Is(err, transport.ErrSessionClosed) {
					return err
				}

				a.logger.Warn("error sending audio to session", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *SessionAgent) outboundTokenLoop(ctx context.Context) error {
	if !a.waitForMessageReady(ctx) {
		return ctx.Err()
	}

	for {
		select {
		case token, ok := <-a.tokenCh:
			if !ok {
				return nil
			}

			text, err := a.options.messageSerializer.Serialize(Message{MessageID: token.MessageID, Role: MessageRoleAgent, Text: token.Text})
			if err != nil {
				a.logger.Warn("error serializing token message", "error", err)
				continue
			}

			if err := a.session.SendMessage(text); err != nil {
				if errors.Is(err, transport.ErrSessionClosed) {
					return err
				}

				if errors.Is(err, transport.ErrMessageNotReady) {
					continue
				}

				a.logger.Warn("error sending token text to session", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *SessionAgent) outboundPromptLoop(ctx context.Context) error {
	if !a.waitForMessageReady(ctx) {
		return ctx.Err()
	}

	for {
		select {
		case prompt, ok := <-a.promptCh:
			if !ok {
				return nil
			}

			text, err := a.options.messageSerializer.Serialize(Message{MessageID: prompt.MessageID, Role: MessageRoleUser, Text: prompt.Text})
			if err != nil {
				a.logger.Warn("error serializing prompt message", "error", err)
				continue
			}

			if err := a.session.SendMessage(text); err != nil {
				if errors.Is(err, transport.ErrSessionClosed) {
					return err
				}

				if errors.Is(err, transport.ErrMessageNotReady) {
					continue
				}

				a.logger.Warn("error sending prompt text to session", "error", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *SessionAgent) waitForMessageReady(ctx context.Context) bool {
	select {
	case <-a.session.MessageReady():
		return true
	case <-ctx.Done():
		return false
	}
}
