package messaging

import "testing"

func TestMarkdownToPlainTextPreservesParagraphBreaks(t *testing.T) {
	input := "First paragraph.\n\nSecond paragraph.\n\n\n\nThird paragraph."
	want := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."

	if got := MarkdownToPlainText(input); got != want {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextPreservesListReadability(t *testing.T) {
	input := "- first item\n- second item\n1. ordered item\n2) another ordered item"
	want := "• first item\n• second item\n1. ordered item\n2. another ordered item"

	if got := MarkdownToPlainText(input); got != want {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextSplitsInlineBulletsAfterColon(t *testing.T) {
	want := "Intro:\n• item one\n• item two"

	for _, input := range []string{
		"Intro: • item one • item two",
		"Intro:• item one• item two",
		"Intro: •item one •item two",
	} {
		if got := MarkdownToPlainText(input); got != want {
			t.Fatalf("MarkdownToPlainText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMarkdownToPlainTextDoesNotCollapseAdjacentLines(t *testing.T) {
	input := "Sentence one.\nSentence two.\n- item one\n- item two"
	want := "Sentence one.\nSentence two.\n• item one\n• item two"

	if got := MarkdownToPlainText(input); got != want {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextConvertsLocalMarkers(t *testing.T) {
	input := "Intro[NEW LINE]Next[PARAGRAPH][ITEM]First[NEW LINE][ITEM]Second[EOF][PARAGRAPH][ITEM]"
	want := "Intro\nNext\n\n• First\n• Second"

	if got := MarkdownToPlainText(input); got != want {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextStripsEOFButKeepsNonMarkerTail(t *testing.T) {
	input := "Before[EOF]after"
	want := "Beforeafter"

	if got := MarkdownToPlainText(input); got != want {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextPreservesEnglishASCIIQuotes(t *testing.T) {
	input := `He said "hello" and it's fine.`

	if got := MarkdownToPlainText(input); got != input {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, input)
	}
}

func TestMarkdownToPlainTextNormalizesEnglishChineseQuotes(t *testing.T) {
	input := "He said “hello” and it’s fine."
	want := `He said "hello" and it's fine.`

	if got := MarkdownToPlainText(input); got != want {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextPreservesChinesePunctuation(t *testing.T) {
	input := "他说：“你好”，这是‘测试’。"

	if got := MarkdownToPlainText(input); got != input {
		t.Fatalf("MarkdownToPlainText() = %q, want %q", got, input)
	}
}

func TestPlainTextReplyChunksKeepsIntroWithFollowingList(t *testing.T) {
	input := "Intro:\n- item one\n- item two"
	want := []string{"Intro:\n• item one\n• item two"}

	assertChunks(t, PlainTextReplyChunks(input), want)
}

func TestPlainTextReplyChunksKeepsIntroWithFollowingListAcrossParagraph(t *testing.T) {
	input := "Intro:\n\n- item one\n- item two"
	want := []string{"Intro:\n• item one\n• item two"}

	assertChunks(t, PlainTextReplyChunks(input), want)
}

func TestPlainTextReplyChunksKeepsOrderedListLines(t *testing.T) {
	input := "1) first\n2. second"
	want := []string{"1. first\n2. second"}

	assertChunks(t, PlainTextReplyChunks(input), want)
}

func TestPlainTextReplyChunksSeparatesParagraphsAroundList(t *testing.T) {
	input := "Before.\n\n- item one\n- item two\n\nAfter."
	want := []string{"Before.", "• item one\n• item two", "After."}

	assertChunks(t, PlainTextReplyChunks(input), want)
}

func assertChunks(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("chunks = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunks[%d] = %q, want %q (all chunks %#v)", i, got[i], want[i], got)
		}
	}
}
