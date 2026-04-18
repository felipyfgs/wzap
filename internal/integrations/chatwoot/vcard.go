package chatwoot

import (
	"fmt"
	"strings"
)

func formatVCard(vcard string) string {
	return formatVCardWithName(vcard, "")
}

func formatVCardWithName(vcard, displayName string) string {
	name := ""
	var phones []string
	for _, line := range splitLines(vcard) {
		if startsWithCI(line, "FN:") {
			name = line[3:]
		} else if startsWithCI(line, "TEL") {
			if idx := lastIndex(line, ":"); idx >= 0 {
				phones = append(phones, line[idx+1:])
			}
		}
	}
	if displayName != "" {
		name = displayName
	}
	if name == "" {
		return vcard
	}
	var sb strings.Builder
	sb.WriteString("*Contato:*\n\n")
	sb.WriteString("_Nome:_ ")
	sb.WriteString(name)
	for i, phone := range phones {
		fmt.Fprintf(&sb, "\n_Número (%d):_ %s", i+1, phone)
	}
	return sb.String()
}

func isVCardContent(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "BEGIN:VCARD")
}

func splitVCards(content string) []string {
	var vcards []string
	lines := strings.Split(content, "\n")
	var current strings.Builder
	for _, line := range lines {
		current.WriteString(line)
		current.WriteString("\n")
		if strings.TrimSpace(line) == "END:VCARD" {
			vcards = append(vcards, current.String())
			current.Reset()
		}
	}
	return vcards
}

func extractVCardName(vcard string) string {
	for _, line := range strings.Split(vcard, "\n") {
		if strings.HasPrefix(line, "FN:") {
			return strings.TrimPrefix(strings.TrimSpace(line), "FN:")
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func startsWithCI(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		sc := s[i]
		pc := prefix[i]
		if sc >= 'a' && sc <= 'z' {
			sc -= 32
		}
		if pc >= 'a' && pc <= 'z' {
			pc -= 32
		}
		if sc != pc {
			return false
		}
	}
	return true
}

func lastIndex(s, sep string) int {
	idx := -1
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
		}
	}
	return idx
}
