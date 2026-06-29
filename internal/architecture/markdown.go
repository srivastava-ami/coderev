package architecture

import (
	"html"
	"strings"
)

// MarkdownToHTML converts a small subset of Markdown to safe HTML.
// Handles ATX headers, bullet lists, fenced code blocks, paragraphs,
// and inline bold/code. Sufficient for architecture documents.
func MarkdownToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var sb strings.Builder
	state := &mdState{}
	for _, raw := range lines {
		processLine(&sb, strings.TrimRight(raw, " \t"), state)
	}
	state.flushPara(&sb)
	state.closeList(&sb)
	if state.inCode {
		sb.WriteString("</code></pre>\n")
	}
	return sb.String()
}

type mdState struct {
	inList bool
	inCode bool
	para   []string
}

func (s *mdState) flushPara(sb *strings.Builder) {
	if len(s.para) == 0 {
		return
	}
	sb.WriteString("<p>")
	sb.WriteString(inlineMarkdown(strings.Join(s.para, " ")))
	sb.WriteString("</p>\n")
	s.para = s.para[:0]
}

func (s *mdState) closeList(sb *strings.Builder) {
	if s.inList {
		sb.WriteString("</ul>\n")
		s.inList = false
	}
}

func processLine(sb *strings.Builder, line string, s *mdState) {
	if s.inCode {
		processCodeLine(sb, line, s)
		return
	}
	if strings.HasPrefix(line, "```") {
		s.flushPara(sb)
		s.closeList(sb)
		lang := strings.TrimPrefix(line, "```")
		sb.WriteString(`<pre><code class="lang-` + html.EscapeString(lang) + `">`)
		s.inCode = true
		return
	}
	if h, rest, ok := matchHeader(line); ok {
		s.flushPara(sb)
		s.closeList(sb)
		sb.WriteString(h + inlineMarkdown(rest) + "</" + h[1:] + "\n")
		return
	}
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		s.flushPara(sb)
		if !s.inList {
			sb.WriteString("<ul>\n")
			s.inList = true
		}
		sb.WriteString("<li>" + inlineMarkdown(line[2:]) + "</li>\n")
		return
	}
	if line == "" {
		s.flushPara(sb)
		s.closeList(sb)
		return
	}
	s.para = append(s.para, line)
}

func processCodeLine(sb *strings.Builder, line string, s *mdState) {
	if strings.HasPrefix(line, "```") {
		sb.WriteString("</code></pre>\n")
		s.inCode = false
	} else {
		sb.WriteString(html.EscapeString(line) + "\n")
	}
}

func matchHeader(line string) (tag, rest string, ok bool) {
	switch {
	case strings.HasPrefix(line, "### "):
		return "<h3>", line[4:], true
	case strings.HasPrefix(line, "## "):
		return "<h2>", line[3:], true
	case strings.HasPrefix(line, "# "):
		return "<h1>", line[2:], true
	}
	return "", "", false
}

// inlineMarkdown handles **bold**, `code`, and HTML escaping.
func inlineMarkdown(s string) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			if end := strings.Index(s[i+2:], "**"); end >= 0 {
				sb.WriteString("<strong>")
				sb.WriteString(html.EscapeString(s[i+2 : i+2+end]))
				sb.WriteString("</strong>")
				i += 2 + end + 2
				continue
			}
		}
		if s[i] == '`' {
			if end := strings.Index(s[i+1:], "`"); end >= 0 {
				sb.WriteString("<code>")
				sb.WriteString(html.EscapeString(s[i+1 : i+1+end]))
				sb.WriteString("</code>")
				i += 1 + end + 1
				continue
			}
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}
