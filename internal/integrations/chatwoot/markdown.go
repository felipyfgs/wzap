package chatwoot

import (
	"regexp"
	"strings"
)

var (
	waBoldToCW   = regexp.MustCompile(`\*([^*\n]+?)\*`)
	waItalicToCW = regexp.MustCompile(`_([^_\n]+?)_`)
	waStrikeToCW = regexp.MustCompile(`~([^~\n]+?)~`)

	cwBoldToWA   = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	cwItalicToWA = regexp.MustCompile(`\*(\S(?:[^*\n]*\S)?)\*`)
	cwStrikeToWA = regexp.MustCompile(`~~([^~\n]+?)~~`)
)

func convertWAToCWMarkdown(s string) string {
	s = waBoldToCW.ReplaceAllString(s, "**${1}**")
	s = waItalicToCW.ReplaceAllString(s, "*${1}*")
	s = waStrikeToCW.ReplaceAllString(s, "~~${1}~~")
	return s
}

func convertCWToWAMarkdown(s string) string {
	s = cwBoldToWA.ReplaceAllString(s, "\x00BOLD\x00${1}\x00/BOLD\x00")
	s = cwStrikeToWA.ReplaceAllString(s, "\x00STRIKE\x00${1}\x00/STRIKE\x00")
	s = cwItalicToWA.ReplaceAllString(s, "_${1}_")
	s = strings.ReplaceAll(s, "\x00BOLD\x00", "*")
	s = strings.ReplaceAll(s, "\x00/BOLD\x00", "*")
	s = strings.ReplaceAll(s, "\x00STRIKE\x00", "~")
	s = strings.ReplaceAll(s, "\x00/STRIKE\x00", "~")
	return s
}
