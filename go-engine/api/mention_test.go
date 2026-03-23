package api

import "testing"

func TestParseMentionAtStart(t *testing.T) {
	alias, clean := ParseMention("@claude viết email")
	if alias != "claude" || clean != "viết email" {
		t.Errorf("got (%q, %q), want ('claude', 'viết email')", alias, clean)
	}
}

func TestParseMentionAtEnd(t *testing.T) {
	alias, clean := ParseMention("viết email @gpt4")
	if alias != "gpt4" || clean != "viết email" {
		t.Errorf("got (%q, %q), want ('gpt4', 'viết email')", alias, clean)
	}
}

func TestParseMentionNoMention(t *testing.T) {
	alias, clean := ParseMention("viết email")
	if alias != "" || clean != "viết email" {
		t.Errorf("got (%q, %q), want ('', 'viết email')", alias, clean)
	}
}

func TestParseMentionMiddle(t *testing.T) {
	alias, clean := ParseMention("hãy @claude viết email")
	if alias != "claude" || clean != "hãy viết email" {
		t.Errorf("got (%q, %q), want ('claude', 'hãy viết email')", alias, clean)
	}
}

func TestParseMentionEmpty(t *testing.T) {
	alias, clean := ParseMention("")
	if alias != "" || clean != "" {
		t.Errorf("got (%q, %q), want ('', '')", alias, clean)
	}
}

func TestParseMentionOnlyAlias(t *testing.T) {
	alias, clean := ParseMention("@claude")
	if alias != "claude" || clean != "" {
		t.Errorf("got (%q, %q), want ('claude', '')", alias, clean)
	}
}

func TestParseMentionEmail(t *testing.T) {
	alias, _ := ParseMention("gửi cho user@example.com")
	if alias != "" {
		t.Errorf("email mistaken as mention: alias=%q", alias)
	}
}
