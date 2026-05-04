package guardgo

import (
	"net/http"
	"strings"
)

func (e *Engine) evaluateRuleset(r *http.Request) int64 {
	compiled := e.compiled
	if e.ruleset != nil {
		compiled = e.ruleset.Current()
		e.compiled = compiled
	}
	if compiled == nil {
		return 0
	}
	var score int64
	if compiled.Any != nil {
		for _, match := range compiled.Any.FindAll(requestAnyText(r)) {
			score += match.Weight
		}
	}
	if compiled.Query != nil && r.URL != nil {
		for _, match := range compiled.Query.FindAll(r.URL.RawQuery) {
			score += match.Weight
		}
	}
	if compiled.Path != nil && r.URL != nil {
		for _, match := range compiled.Path.FindAll(r.URL.Path) {
			score += match.Weight
		}
	}
	if compiled.Headers != nil {
		for _, match := range compiled.Headers.FindAll(headersText(r.Header)) {
			score += match.Weight
		}
	}
	return score
}

func requestAnyText(r *http.Request) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(256)
	b.WriteString(r.Method)
	b.WriteByte(' ')
	if r.URL != nil {
		b.WriteString(r.URL.Path)
		b.WriteByte('?')
		b.WriteString(r.URL.RawQuery)
	}
	b.WriteByte(' ')
	b.WriteString(headersText(r.Header))
	return b.String()
}

func headersText(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	var b strings.Builder
	for key, values := range h {
		b.WriteString(key)
		b.WriteByte(':')
		for _, value := range values {
			b.WriteString(value)
			b.WriteByte(';')
		}
		b.WriteByte('\n')
	}
	return b.String()
}
