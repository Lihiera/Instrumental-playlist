package instrumental

import "testing"

func TestSelectTargetRuleOneWinsOverKaraoke(t *testing.T) {
	original := Track{Title: "Original Song", URL: "https://open.spotify.com/track/original", Artists: []string{"Artist One"}}
	candidates := []Candidate{
		{Track: Track{Title: "Original Song カラオケ", Artists: []string{"Someone Else"}, URL: "https://open.spotify.com/track/karaoke"}, URI: "spotify:track:karaoke"},
		{Track: Track{Title: "Original Song - Instrumental", Artists: []string{"Artist One"}, URL: "https://open.spotify.com/track/instrumental"}, URI: "spotify:track:instrumental"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found {
		t.Fatal("SelectTarget did not find a target")
	}
	if got.Target.URI != "spotify:track:instrumental" {
		t.Fatalf("target URI = %q", got.Target.URI)
	}
}

func TestSelectTargetUsesKaraokeOnlyWhenNoRuleOneMatch(t *testing.T) {
	original := Track{Title: "Original Song", Artists: []string{"Artist One"}}
	candidates := []Candidate{
		{Track: Track{Title: "Original Song - Instrumental", Artists: []string{"Different Artist"}}, URI: "spotify:track:not-match"},
		{Track: Track{Title: "Original Song カラオケ", Artists: []string{"Different Artist"}}, URI: "spotify:track:karaoke"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found {
		t.Fatal("SelectTarget did not find a target")
	}
	if got.Target.URI != "spotify:track:karaoke" {
		t.Fatalf("target URI = %q", got.Target.URI)
	}
}

func TestSelectTargetDoesNotUseKaraokeSearchResultWhenCandidateTitleOmitsKaraoke(t *testing.T) {
	original := Track{Title: "アンコール", Artists: []string{"YOASOBI"}}
	candidates := []Candidate{
		{Track: Track{Title: "アンコール", Artists: []string{"Karaoke Artist"}}, URI: "spotify:track:karaoke"},
	}

	got := SelectTarget(original, candidates)

	if got.Found {
		t.Fatalf("SelectTarget unexpectedly found a karaoke target without keyword: %+v", got)
	}
}

func TestSelectTargetReturnsSafeNotFoundTrack(t *testing.T) {
	original := Track{Title: " Missing Song ", URL: " https://open.spotify.com/track/missing ", Artists: []string{"Artist One"}}

	got := SelectTarget(original, nil)

	if got.Found {
		t.Fatal("SelectTarget found an unexpected target")
	}
	if got.NotFound.Title != "Missing Song" || got.NotFound.URL != "https://open.spotify.com/track/missing" {
		t.Fatalf("not found = %+v", got.NotFound)
	}
	if len(got.NotFound.Artists) != 0 {
		t.Fatalf("not found exposed artists: %+v", got.NotFound)
	}
}

func TestSelectTargetMatchesAnySharedArtist(t *testing.T) {
	original := Track{Title: "Collab Song", Artists: []string{"Artist One", "Artist Two"}}
	candidates := []Candidate{
		{Track: Track{Title: "Collab Song インスト", Artists: []string{"Artist Two", "Other Artist"}}, URI: "spotify:track:target"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found || got.Target.URI != "spotify:track:target" {
		t.Fatalf("selection = %+v", got)
	}
}

func TestSelectTargetNormalizesEnglishCaseAndWhitespace(t *testing.T) {
	original := Track{Title: "  Original   Song ", Artists: []string{"Artist One"}}
	candidates := []Candidate{
		{Track: Track{Title: "original song - INSTRUMENTAL", Artists: []string{" artist   one "}}, URI: "spotify:track:target"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found || got.Target.URI != "spotify:track:target" {
		t.Fatalf("selection = %+v", got)
	}
}

func TestSelectTargetComparesOriginalTitleBeforeParentheses(t *testing.T) {
	original := Track{Title: "再会 (produced by Ayase)", Artists: []string{"LiSA"}}
	candidates := []Candidate{
		{Track: Track{Title: "再会 カラオケ", Artists: []string{"Karaoke Artist"}}, URI: "spotify:track:karaoke"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found || got.Target.URI != "spotify:track:karaoke" {
		t.Fatalf("selection = %+v", got)
	}
}

func TestSelectTargetComparesOriginalTitleBeforeFullWidthParentheses(t *testing.T) {
	original := Track{Title: "おもかげ（produced by Vaundy）", Artists: []string{"milet"}}
	candidates := []Candidate{
		{Track: Track{Title: "おもかげ Instrumental", Artists: []string{"milet"}}, URI: "spotify:track:instrumental"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found || got.Target.URI != "spotify:track:instrumental" {
		t.Fatalf("selection = %+v", got)
	}
}

func TestSelectTargetDoesNotTrimCandidateTitleParentheses(t *testing.T) {
	original := Track{Title: "Song Extra", Artists: []string{"Artist One"}}
	candidates := []Candidate{
		{Track: Track{Title: "Song (Extra) Instrumental", Artists: []string{"Artist One"}}, URI: "spotify:track:wrong"},
	}

	got := SelectTarget(original, candidates)

	if got.Found {
		t.Fatalf("SelectTarget unexpectedly matched candidate with trimmed title: %+v", got)
	}
}

func TestSelectTargetMatchesJapaneseKeywords(t *testing.T) {
	original := Track{Title: "日本語の曲", Artists: []string{"歌手"}}
	candidates := []Candidate{
		{Track: Track{Title: "日本語の曲 インスト", Artists: []string{"歌手"}}, URI: "spotify:track:instrumental"},
	}

	got := SelectTarget(original, candidates)

	if !got.Found || got.Target.URI != "spotify:track:instrumental" {
		t.Fatalf("selection = %+v", got)
	}
}
