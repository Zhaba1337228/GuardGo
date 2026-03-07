package rules

import (
	"net/http"
	"strings"
)

// PathContains returns threat weight when URL path contains any listed token.
type PathContains struct {
	Tokens []string
	Weight int
}

func (r PathContains) Evaluate(req *http.Request) int {
	if req == nil || req.URL == nil || len(r.Tokens) == 0 {
		return 0
	}
	path := strings.ToLower(req.URL.Path)
	for _, token := range r.Tokens {
		if token != "" && strings.Contains(path, strings.ToLower(token)) {
			return weightOrDefault(r.Weight)
		}
	}
	return 0
}

// QueryContains returns threat weight when raw query contains any listed token.
type QueryContains struct {
	Tokens []string
	Weight int
}

func (r QueryContains) Evaluate(req *http.Request) int {
	if req == nil || req.URL == nil || len(r.Tokens) == 0 {
		return 0
	}
	rawQuery := strings.ToLower(req.URL.RawQuery)
	for _, token := range r.Tokens {
		if token != "" && strings.Contains(rawQuery, strings.ToLower(token)) {
			return weightOrDefault(r.Weight)
		}
	}
	return 0
}

// HeaderContains returns threat weight when a header contains any listed token.
type HeaderContains struct {
	Name   string
	Tokens []string
	Weight int
}

func (r HeaderContains) Evaluate(req *http.Request) int {
	if req == nil || r.Name == "" || len(r.Tokens) == 0 {
		return 0
	}
	values := req.Header.Values(r.Name)
	for _, value := range values {
		lowerValue := strings.ToLower(value)
		for _, token := range r.Tokens {
			if token != "" && strings.Contains(lowerValue, strings.ToLower(token)) {
				return weightOrDefault(r.Weight)
			}
		}
	}
	return 0
}

func weightOrDefault(weight int) int {
	if weight <= 0 {
		return 1
	}
	return weight
}
