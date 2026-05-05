package helper

func SplitAtSentenceEnd(text string, sentenceEndRunes []rune) (before string, after string, found bool) {
	runes := []rune(text)
	lastIdx := -1

	for i, r := range runes {
		for _, end := range sentenceEndRunes {
			if r == end {
				lastIdx = i
				break
			}
		}
	}

	if lastIdx < 0 {
		return "", "", false
	}

	return string(runes[:lastIdx+1]), string(runes[lastIdx+1:]), true
}
