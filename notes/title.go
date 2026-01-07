package notes

import "strings"

const defaultTitle = "Untitled"

// NormalizeTitle trims leading hashes and whitespace from a title string.
func NormalizeTitle(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultTitle
	}

	trimmed = strings.TrimLeft(trimmed, "#")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return defaultTitle
	}

	return trimmed
}

// NormalizeTitleFromBody pulls the first non-empty line and strips heading markers.
func NormalizeTitleFromBody(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			trimmed = strings.TrimLeft(trimmed, "#")
			trimmed = strings.TrimSpace(trimmed)
			if trimmed == "" {
				continue
			}
		}

		return trimmed
	}

	return defaultTitle
}
