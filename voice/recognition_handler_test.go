package voice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCombineTranscripts_Empty(t *testing.T) {
	assert.Equal(t, "", combineTranscripts(nil))
}

func TestCombineTranscripts_Single(t *testing.T) {
	assert.Equal(t, "hello", combineTranscripts([]Transcript{{Text: "hello"}}))
}

func TestCombineTranscripts_Multiple(t *testing.T) {
	result := combineTranscripts([]Transcript{
		{Text: "hello"},
		{Text: "world"},
	})
	assert.Equal(t, "hello world", result)
}

func TestCombineTranscripts_Three(t *testing.T) {
	result := combineTranscripts([]Transcript{
		{Text: "one"},
		{Text: "two"},
		{Text: "three"},
	})
	assert.Equal(t, "one two three", result)
}
