package api

import (
	"regexp"
	"strings"
)

// mentionRegex match @alias — không match email (ký tự word trước @)
var mentionRegex = regexp.MustCompile(`(?:^|\s)@([a-zA-Z0-9_-]+)`)

// ParseMention tách @alias và prompt sạch
func ParseMention(prompt string) (alias, cleanPrompt string) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", ""
	}

	loc := mentionRegex.FindStringSubmatchIndex(prompt)
	if loc == nil {
		return "", prompt
	}

	alias = strings.ToLower(prompt[loc[2]:loc[3]])

	// Xóa match khỏi prompt
	cleanPrompt = prompt[:loc[0]] + prompt[loc[1]:]
	cleanPrompt = strings.TrimSpace(cleanPrompt)
	cleanPrompt = strings.Join(strings.Fields(cleanPrompt), " ")

	return alias, cleanPrompt
}
