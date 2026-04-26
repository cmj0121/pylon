package pylon

import (
	"regexp"
	"strconv"
	"strings"
)

// parseFrontmatter parses the YAML-subset body between `---` fences.
// Supports `size:`, `theme:`, and `data:` (flat list or map-of-series).
// Any unsupported `data:` shape is recorded as a toast-path error.
//
// src is a sub-Source covering the inner bytes between the fences;
// absolute positions on Meta spans resolve via src's base + linePrefix.
func parseFrontmatter(src *Source) Meta {
	meta := Meta{}
	errs := []MetaError{}
	lines, starts := splitLinesWithOffsets(src.View())

	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}
		if dataHeaderRe.MatchString(line) {
			// Collect indented continuation until EOF or next top-level key.
			section := []string{}
			j := i + 1
			for j < len(lines) {
				l := lines[j]
				if strings.TrimSpace(l) == "" {
					section = append(section, l)
					j++
					continue
				}
				if topLevelKeyRe.MatchString(l) {
					break
				}
				section = append(section, l)
				j++
			}
			// Span covers the header line through the end of the last
			// accumulated section line. When the section has no lines,
			// it collapses to the header line itself.
			lastLine := j - 1
			if lastLine < i {
				lastLine = i
			}
			dataSpan := src.SpanOf(starts[i], starts[lastLine]+len(lines[lastLine]))
			val, ok := parseDataSection(section)
			if ok {
				meta.Data = val
				meta.DataSpan = dataSpan
			} else {
				errs = append(errs, MetaError{
					Message: "Unsupported data: frontmatter shape",
					Span:    dataSpan,
				})
			}
			i = j
			continue
		}
		m := keyValueRe.FindStringSubmatch(line)
		if m == nil {
			i++
			continue
		}
		key, raw := m[1], m[2]
		lineSpan := src.SpanOf(starts[i], starts[i]+len(lines[i]))
		switch key {
		case "size":
			sm := sizeRe.FindStringSubmatch(raw)
			if sm != nil {
				w, _ := strconv.Atoi(sm[1])
				h, _ := strconv.Atoi(sm[2])
				meta.Size = &Size{W: w, H: h}
				meta.SizeSpan = lineSpan
			}
		case "theme":
			meta.Theme = raw
			meta.ThemeSpan = lineSpan
		case "color":
			// Accept lowercase `true` / `false`. Anything else leaves
			// Color nil (no preference) so the CLI falls through to
			// auto-mode heuristics — noisy values are silently ignored
			// rather than producing a toast, matching how `theme:` and
			// `size:` handle malformed values.
			switch raw {
			case "true":
				v := true
				meta.Color = &v
				meta.ColorSpan = lineSpan
			case "false":
				v := false
				meta.Color = &v
				meta.ColorSpan = lineSpan
			}
		case "banner":
			// Recognized fonts: `monospace`, `digital`, `mini`. Any
			// other value falls through to the theme-driven default
			// — same silent-ignore policy as `theme:`, `size:`, and
			// `color:`. Per-box `[ ... | banner:FONT ]` overrides this
			// frontmatter default at render time.
			switch raw {
			case "monospace", "digital", "mini":
				meta.Banner = raw
				meta.BannerSpan = lineSpan
			}
		}
		i++
	}
	if len(errs) > 0 {
		meta.Errors = errs
	}
	return meta
}

var (
	dataHeaderRe   = regexp.MustCompile(`^\s*data\s*:\s*$`)
	topLevelKeyRe  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\s*:`)
	keyValueRe     = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*?)\s*$`)
	sizeRe         = regexp.MustCompile(`^(\d+)\s*[xX]\s*(\d+)$`)
	yamlNumberRe   = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
	yamlKeyValueRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$`)
	yamlEmptyKeyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*$`)
)

// parseYamlScalar returns the scalar value and a success flag. Numbers
// become float64; quoted strings become the inner value (no escape
// expansion); `[n, n, ...]` flow arrays become []float64 (heatmap's
// row shape); unquoted runs stay as trimmed literals.
func parseYamlScalar(raw string) (interface{}, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, false
	}
	if yamlNumberRe.MatchString(s) {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, false
		}
		return f, true
	}
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return parseFlowNumberArray(s[1 : len(s)-1])
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		if strings.Contains(inner, "\"") || strings.Contains(inner, "\\") {
			return nil, false
		}
		return inner, true
	}
	return s, true
}

// parseFlowNumberArray reads the comma-separated body of a flow
// array (`[1, 2, 3]` → "1, 2, 3"). Only numbers are supported — the
// heatmap shape is the sole v1 consumer. Empty brackets `[]` parse
// to an empty []float64.
func parseFlowNumberArray(body string) ([]float64, bool) {
	s := strings.TrimSpace(body)
	if s == "" {
		return []float64{}, true
	}
	parts := strings.Split(s, ",")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		tok := strings.TrimSpace(p)
		if !yamlNumberRe.MatchString(tok) {
			return nil, false
		}
		f, err := strconv.ParseFloat(tok, 64)
		if err != nil {
			return nil, false
		}
		out = append(out, f)
	}
	return out, true
}

// parseDataSection splits into flat-list vs named-series based on the
// first non-blank line's shape. Returns (value, ok). Tabs in any
// leading indent reject the whole parse.
func parseDataSection(lines []string) (interface{}, bool) {
	first := -1
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			first = i
			break
		}
	}
	if first == -1 {
		return nil, true
	}
	for _, l := range lines {
		lead := leadingWhitespace(l)
		if strings.Contains(lead, "\t") {
			return nil, false
		}
	}
	baseIndent := leadingSpaces(lines[first])
	if baseIndent == 0 {
		return nil, true
	}
	firstContent := lines[first][baseIndent:]
	if strings.HasPrefix(firstContent, "- ") || firstContent == "-" {
		return parseSeriesList(lines, baseIndent)
	}
	if topLevelKeyRe.MatchString(firstContent) {
		return parseSeriesMap(lines, baseIndent)
	}
	return nil, false
}

func parseSeriesList(lines []string, baseIndent int) (interface{}, bool) {
	items := []map[string]interface{}{}
	var current map[string]interface{}
	childIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lead := leadingSpaces(line)
		if lead < baseIndent {
			return nil, false
		}
		content := line[baseIndent:]
		if lead == baseIndent {
			if !strings.HasPrefix(content, "- ") && content != "-" {
				return nil, false
			}
			current = map[string]interface{}{}
			items = append(items, current)
			var rest string
			if content == "-" {
				rest = ""
			} else {
				rest = content[2:]
			}
			if rest != "" {
				m := yamlKeyValueRe.FindStringSubmatch(rest)
				if m == nil {
					return nil, false
				}
				v, ok := parseYamlScalar(m[2])
				if !ok {
					return nil, false
				}
				current[m[1]] = v
				if childIndent == -1 {
					childIndent = baseIndent + 2
				}
			}
		} else {
			if current == nil {
				return nil, false
			}
			if childIndent == -1 {
				childIndent = lead
			}
			if lead != childIndent {
				return nil, false
			}
			after := content[lead-baseIndent:]
			kv := yamlKeyValueRe.FindStringSubmatch(after)
			if kv == nil {
				return nil, false
			}
			v, ok := parseYamlScalar(kv[2])
			if !ok {
				return nil, false
			}
			current[kv[1]] = v
		}
	}
	return items, true
}

func parseSeriesMap(lines []string, baseIndent int) (interface{}, bool) {
	out := map[string]interface{}{}
	currentKey := ""
	var childLines []string
	childIndent := -1
	flushChild := func() bool {
		if currentKey == "" {
			return true
		}
		if len(childLines) == 0 {
			return false
		}
		sub, ok := parseSeriesList(childLines, childIndent)
		if !ok {
			return false
		}
		out[currentKey] = sub
		currentKey = ""
		childLines = nil
		childIndent = -1
		return true
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if childLines != nil {
				childLines = append(childLines, line)
			}
			continue
		}
		lead := leadingSpaces(line)
		if lead < baseIndent {
			return nil, false
		}
		if lead == baseIndent {
			if !flushChild() {
				return nil, false
			}
			content := line[baseIndent:]
			m := yamlEmptyKeyRe.FindStringSubmatch(content)
			if m == nil {
				return nil, false
			}
			currentKey = m[1]
			childLines = []string{}
		} else {
			if currentKey == "" {
				return nil, false
			}
			if childIndent == -1 {
				childIndent = lead
			}
			if lead < childIndent {
				return nil, false
			}
			childLines = append(childLines, line)
		}
	}
	if !flushChild() {
		return nil, false
	}
	return out, true
}

// splitLinesWithOffsets splits s on `\n`, stripping a trailing `\r`
// from each returned content line (so regex matches behave exactly as
// they did under the earlier `\r\n` → `\n` normalisation), while
// retaining per-line starting byte offsets into the ORIGINAL s. The
// two slices share length; starts[i] points to the first byte of
// lines[i] before any CR-stripping.
func splitLinesWithOffsets(s string) (content []string, starts []int) {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			content = append(content, line)
			starts = append(starts, start)
			start = i + 1
		}
	}
	tail := s[start:]
	if len(tail) > 0 && tail[len(tail)-1] == '\r' {
		tail = tail[:len(tail)-1]
	}
	content = append(content, tail)
	starts = append(starts, start)
	return
}

// leadingSpaces returns the count of leading ' ' runes. Non-space
// whitespace (tabs, etc.) halts the count.
func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

// leadingWhitespace returns the maximal leading \s run (spaces, tabs,
// etc.) — used to detect tab-indented lines and reject them.
func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}
