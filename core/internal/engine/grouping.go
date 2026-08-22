package engine

import (
	"regexp"
	"strings"
)

// CountryFromName extracts a 2-letter country code from a server/subscription name.
// Looks for patterns like "US | Server", "[DE] Server", or prefix "US-Server".
func CountryFromName(name string) string {
	// Pattern 1: Starts with 2 uppercase letters followed by space, dash or bracket
	prefixRe := regexp.MustCompile(`^([A-Z]{2})[\s\-]`)
	if matches := prefixRe.FindStringSubmatch(name); len(matches) > 1 {
		return matches[1]
	}

	// Pattern 2: [XX] somewhere in the name
	bracketRe := regexp.MustCompile(`\[([A-Z]{2})\]`)
	if matches := bracketRe.FindStringSubmatch(name); len(matches) > 1 {
		return matches[1]
	}

	// Pattern 3: " | XX" or " - XX" separator
	sepRe := regexp.MustCompile(`[\s|\-]+\s*([A-Z]{2})(?:\s|$)`)
	if matches := sepRe.FindStringSubmatch(name); len(matches) > 1 {
		return matches[1]
	}

	// Default: return empty if no pattern matches
	return ""
}

// FlagEmoji converts a 2-letter country code to a flag emoji.
// Uses Unicode regional indicator symbols (U+1F1E6 + offset).
func FlagEmoji(cc string) string {
	if len(cc) != 2 {
		return ""
	}
	cc = strings.ToUpper(cc)
	
	var result strings.Builder
	for _, r := range cc {
		if r >= 'A' && r <= 'Z' {
			// Regional Indicator Symbol Letter
			result.WriteRune(0x1F1E6 + (r - 'A'))
		}
	}
	return result.String()
}

// CountryName returns a human-readable country name from code (optional helper).
func CountryName(cc string) string {
	// Minimal mapping; can be extended
	countries := map[string]string{
		"US": "United States",
		"GB": "United Kingdom",
		"DE": "Germany",
		"FR": "France",
		"NL": "Netherlands",
		"RU": "Russia",
		"CN": "China",
		"JP": "Japan",
		"KR": "South Korea",
		"SG": "Singapore",
		"UA": "Ukraine",
		"PL": "Poland",
		"FI": "Finland",
		"SE": "Sweden",
		"NO": "Norway",
		"CA": "Canada",
		"AU": "Australia",
		"BR": "Brazil",
		"IN": "India",
		"TR": "Turkey",
		"IR": "Iran",
	}
	if name, ok := countries[strings.ToUpper(cc)]; ok {
		return name
	}
	return cc
}
