package helper_test

import (
	"testing"

	"github.com/josephnhtam/live-agent-go/voice/helper"
	"github.com/stretchr/testify/assert"
)

func TestSplitAtSentenceEnd(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		delimiters []rune
		wantBefore string
		wantAfter  string
		wantFound  bool
	}{
		{
			name:       "period at end",
			text:       "Hello world.",
			delimiters: []rune{'.', '!', '?'},
			wantBefore: "Hello world.",
			wantAfter:  "",
			wantFound:  true,
		},
		{
			name:       "period in middle",
			text:       "Hello. World",
			delimiters: []rune{'.'},
			wantBefore: "Hello.",
			wantAfter:  " World",
			wantFound:  true,
		},
		{
			name:       "multiple delimiters picks last",
			text:       "One. Two! Three",
			delimiters: []rune{'.', '!'},
			wantBefore: "One. Two!",
			wantAfter:  " Three",
			wantFound:  true,
		},
		{
			name:       "no match",
			text:       "Hello world",
			delimiters: []rune{'.', '!'},
			wantBefore: "",
			wantAfter:  "",
			wantFound:  false,
		},
		{
			name:       "empty text",
			text:       "",
			delimiters: []rune{'.'},
			wantBefore: "",
			wantAfter:  "",
			wantFound:  false,
		},
		{
			name:       "unicode delimiter",
			text:       "你好。世界",
			delimiters: []rune{'。'},
			wantBefore: "你好。",
			wantAfter:  "世界",
			wantFound:  true,
		},
		{
			name:       "single char is delimiter",
			text:       ".",
			delimiters: []rune{'.'},
			wantBefore: ".",
			wantAfter:  "",
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, after, found := helper.SplitAtSentenceEnd(tt.text, tt.delimiters)
			assert.Equal(t, tt.wantBefore, before)
			assert.Equal(t, tt.wantAfter, after)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}
