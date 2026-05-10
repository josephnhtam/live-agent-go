package helper

func SplitAtSentenceEnd(text string, sentenceEndRunes []rune) (before string, after string, found bool) {
	runes := []rune(text)
	firstIdx := -1

	for i, r := range runes {
		for _, end := range sentenceEndRunes {
			if r == end {
				firstIdx = i
				break
			}
		}

		if firstIdx != -1 {
			break
		}
	}

	if firstIdx < 0 {
		return "", "", false
	}

	return string(runes[:firstIdx+1]), string(runes[firstIdx+1:]), true
}
