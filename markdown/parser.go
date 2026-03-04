package markdown

import (
	"bufio"
	"strings"
)

// ParseResult holds the parsed components of a Markdown file.
type ParseResult struct {
	FrontMatter map[string]string // raw key-value pairs from YAML front matter
	Title       string            // first H1 heading, or empty
	Content     string            // body text after front matter
	Tags        []string          // values from the "tags" front matter key
	Links       []string          // [[wiki-link]] targets found in the body
}

// Parse splits raw Markdown content into front matter and body,
// then extracts the title, tags, and wiki-links.
func Parse(raw string) ParseResult {
	fm, body := splitFrontMatter(raw)
	return ParseResult{
		FrontMatter: fm,
		Title:       extractTitle(body),
		Content:     body,
		Tags:        parseTags(fm["tags"]),
		Links:       extractLinks(body),
	}
}

// splitFrontMatter separates YAML front matter (delimited by "---") from the body.
// Returns an empty map and the full text when no front matter is found.
func splitFrontMatter(raw string) (map[string]string, string) {
	fm := make(map[string]string)

	if !strings.HasPrefix(raw, "---") {
		return fm, raw
	}

	// Find the closing "---"
	rest := raw[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, raw
	}

	yamlBlock := strings.TrimSpace(rest[:idx])
	body := strings.TrimSpace(rest[idx+4:]) // skip "\n---"

	scanner := bufio.NewScanner(strings.NewReader(yamlBlock))
	for scanner.Scan() {
		line := scanner.Text()
		k, v, ok := strings.Cut(line, ":")
		if ok {
			fm[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}

	return fm, body
}

// extractTitle returns the text of the first H1 line, or empty string.
func extractTitle(body string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// parseTags parses a YAML inline sequence like "[Go, CLI]" or "Go, CLI" into a slice.
func parseTags(raw string) []string {
	raw = strings.Trim(raw, "[]")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// extractLinks finds all [[wiki-link]] targets in text.
func extractLinks(body string) []string {
	var links []string
	remaining := body
	for {
		open := strings.Index(remaining, "[[")
		if open < 0 {
			break
		}
		close := strings.Index(remaining[open:], "]]")
		if close < 0 {
			break
		}
		target := remaining[open+2 : open+close]
		// Handle [[page|alias]] — keep only the page part
		if pipe := strings.Index(target, "|"); pipe >= 0 {
			target = target[:pipe]
		}
		links = append(links, strings.TrimSpace(target))
		remaining = remaining[open+close+2:]
	}
	return links
}
