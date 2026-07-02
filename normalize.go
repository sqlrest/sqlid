package sqlid

import (
	"fmt"
	"regexp"
	"strings"
)

// Normalization regexes. The comment and string-literal patterns are RE2
// translations of the historical Python patterns, verified equivalent across
// an edge-case battery (doubled quotes, unterminated strings, multi-line
// comments, optimizer hints).
var (
	// commentRe matches C-style comments that are not optimizer hints (/*+ ... */).
	commentRe = regexp.MustCompile(`/\*[^+].*?\*/`)
	// whitespaceRe matches any run of whitespace.
	whitespaceRe = regexp.MustCompile(`[ \t\n\r\v\f]+`)
	// semicolonRe matches a trailing semicolon and any whitespace after it.
	semicolonRe = regexp.MustCompile(`;[ \t\n\r\v\f]*$`)
	// withRe matches a WITH-clause alias (with x as / , x as) at a segment start.
	withRe = regexp.MustCompile(`(?i)^(?:with\s+|,\s*)([^\s.]+)\s+as`)
	// stringRe matches a single-quoted literal, tolerating doubled single quotes.
	stringRe = regexp.MustCompile(`'(?:''|[^'])*'?`)
	// numberRe matches a whitespace-delimited integer literal.
	numberRe = regexp.MustCompile(`\s\d+\s`)
)

// transform is a single normalization step over a statement.
type transform func(Statement) Statement

func lower(s Statement) Statement {
	return Statement(strings.ToLower(string(s)))
}

func uncomment(s Statement) Statement {
	return Statement(commentRe.ReplaceAllString(string(s), ""))
}

func stripSemicolon(s Statement) Statement {
	return Statement(semicolonRe.ReplaceAllString(string(s), ""))
}

func collapse(s Statement) Statement {
	return Statement(strings.TrimSpace(whitespaceRe.ReplaceAllString(string(s), " ")))
}

func appendNewline(s Statement) Statement {
	return s + "\n"
}

func stripConstants(s Statement) Statement {
	noStrings := stringRe.ReplaceAllString(string(s), "?")
	return Statement(numberRe.ReplaceAllString(noStrings, " ? "))
}

// renameWithAliases replaces each top-level WITH alias with a positional
// ^NNNN^ token, so that equivalent queries with differently named CTEs collapse.
func renameWithAliases(s Statement) Statement {
	stmt := string(s)
	if !withRe.MatchString(stmt) {
		return s
	}
	sequence := 1
	for _, segment := range topLevelSegments(s) {
		match := withRe.FindStringSubmatch(segment)
		if match == nil {
			continue
		}
		stmt = strings.ReplaceAll(stmt, match[1]+" ", fmt.Sprintf("^%04X^ ", sequence))
		sequence++
	}
	return Statement(stmt)
}

// scan accumulates the top-level (depth-zero) text segments of a statement
// while tracking parenthesis depth and skipping quoted regions. It is an
// immutable value advanced character by character: each step returns the next
// scanner state instead of mutating the receiver.
type scan struct {
	segs  []string
	depth int
	start int
	quote byte
}

// inQuote reports whether the current character lies inside a quoted region,
// returning the scanner with the active quote toggled when the character is a
// quote delimiter.
func (s scan) inQuote(c byte) (scan, bool) {
	if c == '\'' || c == '"' {
		s = s.toggle(c)
	}
	return s, s.quote != 0
}

// toggle opens a quoted region on the first delimiter and closes it on a
// matching delimiter; a different delimiter inside a region is left untouched.
func (s scan) toggle(c byte) scan {
	switch s.quote {
	case 0:
		s.quote = c
	case c:
		s.quote = 0
	}
	return s
}

// open records the depth-zero text before a parenthesis and descends a level.
func (s scan) open(index int, stmt string) scan {
	if s.depth == 0 {
		s.segs = append(s.segs, stmt[s.start:index])
	}
	s.depth++
	s.start = index + 1
	return s
}

// closeGroup ascends a level (when one is open) past a parenthesis.
func (s scan) closeGroup(index int) scan {
	if s.depth > 0 {
		s.depth--
	}
	s.start = index + 1
	return s
}

// paren dispatches parenthesis handling for the character at index.
func (s scan) paren(index int, c byte, stmt string) scan {
	switch c {
	case '(':
		return s.open(index, stmt)
	case ')':
		return s.closeGroup(index)
	}
	return s
}

// finish appends the trailing depth-zero segment, dropping the final character
// (the appended newline), matching the historical parser the rewrite relies on.
func (s scan) finish(stmt string) []string {
	if s.start != len(stmt) {
		s.segs = append(s.segs, stmt[s.start:len(stmt)-1])
	}
	return s.segs
}

// topLevelSegments returns the depth-zero text segments of the statement.
func topLevelSegments(s Statement) []string {
	stmt := string(s)
	sc := scan{}
	for index := range len(stmt) {
		next, quoted := sc.inQuote(stmt[index])
		sc = next
		if quoted {
			continue
		}
		sc = sc.paren(index, stmt[index], stmt)
	}
	return sc.finish(stmt)
}
