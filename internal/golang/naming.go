package golang

import (
	"strings"
	"unicode"
)

var commonInitialisms = map[string]bool{
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"UUID":  true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
	"CVV":   true,
}

// SetAdditionalInitialisms adds custom initialisms to the naming rules.
// This should be called once during initialization before generation.
func SetAdditionalInitialisms(initialisms []string) {
	for _, init := range initialisms {
		upper := strings.ToUpper(init)
		commonInitialisms[upper] = true
	}
}

func PascalCase(s string) string {
	words := splitWords(s)
	var result strings.Builder
	for _, word := range words {
		upper := strings.ToUpper(word)
		if commonInitialisms[upper] {
			result.WriteString(upper)
		} else {
			result.WriteString(capitalize(word))
		}
	}
	return result.String()
}

func CamelCase(s string) string {
	words := splitWords(s)
	var result strings.Builder
	for i, word := range words {
		if i == 0 {
			result.WriteString(strings.ToLower(word))
		} else {
			upper := strings.ToUpper(word)
			if commonInitialisms[upper] {
				result.WriteString(upper)
			} else {
				result.WriteString(capitalize(word))
			}
		}
	}
	return result.String()
}

func SnakeCase(s string) string {
	words := splitWords(s)
	for i, word := range words {
		words[i] = strings.ToLower(word)
	}
	return strings.Join(words, "_")
}

func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' || r == '.' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}

		if unicode.IsUpper(r) && i > 0 {
			prev := rune(s[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				if current.Len() > 0 {
					words = append(words, current.String())
					current.Reset()
				}
			}
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}

func ToGoIdentifier(s string) string {
	result := PascalCase(s)
	if len(result) == 0 {
		return "X"
	}
	first := rune(result[0])
	if unicode.IsDigit(first) {
		return "X" + result
	}
	return result
}

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func EscapeKeyword(s string) string {
	lower := strings.ToLower(s)
	if goKeywords[lower] {
		return s + "_"
	}
	return s
}
