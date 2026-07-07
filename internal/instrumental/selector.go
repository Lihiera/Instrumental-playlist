package instrumental

import "strings"

type Track struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Artists []string `json:"artists,omitempty"`
}

type Candidate struct {
	Track
	URI string
}

type Selection struct {
	Found    bool
	Target   Candidate
	NotFound Track
}

func SelectTarget(original Track, candidates []Candidate) Selection {
	if target, ok := firstInstrumentalMatch(original, candidates); ok {
		return Selection{Found: true, Target: target}
	}
	if target, ok := firstKaraokeMatch(original, candidates); ok {
		return Selection{Found: true, Target: target}
	}
	return Selection{NotFound: Track{Title: strings.TrimSpace(original.Title), URL: strings.TrimSpace(original.URL)}}
}

func firstInstrumentalMatch(original Track, candidates []Candidate) (Candidate, bool) {
	for _, candidate := range candidates {
		if titleContainsOriginal(original.Title, candidate.Title) &&
			containsAny(candidate.Title, "instrumental", "インスト") &&
			sharesArtist(original.Artists, candidate.Artists) {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func firstKaraokeMatch(original Track, candidates []Candidate) (Candidate, bool) {
	for _, candidate := range candidates {
		if titleContainsOriginal(original.Title, candidate.Title) && containsAny(candidate.Title, "カラオケ", "karaoke") {
			return candidate, true
		}
	}
	return Candidate{}, false
}

func titleContainsOriginal(originalTitle, candidateTitle string) bool {
	original := normalizeMatchText(NormalizeOriginalTitle(originalTitle))
	candidate := normalizeMatchText(candidateTitle)
	return original != "" && strings.Contains(candidate, original)
}

func containsAny(value string, keywords ...string) bool {
	normalized := normalizeMatchText(value)
	for _, keyword := range keywords {
		if strings.Contains(normalized, normalizeMatchText(keyword)) {
			return true
		}
	}
	return false
}

func sharesArtist(originalArtists, candidateArtists []string) bool {
	original := map[string]struct{}{}
	for _, artist := range originalArtists {
		name := normalizeMatchText(artist)
		if name != "" {
			original[name] = struct{}{}
		}
	}
	if len(original) == 0 {
		return false
	}
	for _, artist := range candidateArtists {
		if _, ok := original[normalizeMatchText(artist)]; ok {
			return true
		}
	}
	return false
}

func normalizeMatchText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func NormalizeOriginalTitle(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(trimTitleQualifier(value))), " ")
}

func trimTitleQualifier(value string) string {
	cut := len(value)
	for _, marker := range []string{"(", "（"} {
		if index := strings.Index(value, marker); index >= 0 && index < cut {
			cut = index
		}
	}
	return value[:cut]
}
