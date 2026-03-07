package guardgo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"guardgo/pkg/dfa"

	"gopkg.in/yaml.v3"
)

var ErrRulesetPathEmpty = errors.New("guardgo: ruleset path is empty")

type MatchTarget string

const (
	MatchQuery   MatchTarget = "query"
	MatchHeaders MatchTarget = "headers"
	MatchPath    MatchTarget = "path"
	MatchAny     MatchTarget = "any"
)

type SignatureRule struct {
	Name           string      `json:"name" yaml:"name"`
	Match          MatchTarget `json:"match" yaml:"match"`
	Pattern        string      `json:"pattern" yaml:"pattern"`
	Weight         int64       `json:"weight" yaml:"weight"`
	CountThreshold int         `json:"count_threshold" yaml:"count_threshold"`
}

type rulesetFile struct {
	Rules []SignatureRule `json:"rules" yaml:"rules"`
}

type CompiledRuleset struct {
	Any     *dfa.Matcher
	Query   *dfa.Matcher
	Headers *dfa.Matcher
	Path    *dfa.Matcher
}

type RulesetStats struct {
	AnyRules    int
	QueryRules  int
	HeaderRules int
	PathRules   int
	AnyNodes    int
	QueryNodes  int
	HeaderNodes int
	PathNodes   int
	TotalRules  int
	TotalNodes  int
}

type RulesetManager struct {
	path string
	cur  atomic.Pointer[CompiledRuleset]
}

func NewRulesetManager(path string) (*RulesetManager, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrRulesetPathEmpty
	}
	manager := &RulesetManager{path: path}
	if err := manager.Reload(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *RulesetManager) Current() *CompiledRuleset {
	return m.cur.Load()
}

func (m *RulesetManager) Reload() error {
	rs, err := LoadRulesetFile(m.path)
	if err != nil {
		return err
	}
	m.cur.Store(rs)
	return nil
}

func (m *RulesetManager) ReloadOnSignal(ctx context.Context, ch <-chan os.Signal, onError func(error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			if err := m.Reload(); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

func LoadRulesetFile(path string) (*CompiledRuleset, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	spec := rulesetFile{}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(body, &spec); err != nil {
			return nil, err
		}
	default:
		if err := json.Unmarshal(body, &spec); err != nil {
			return nil, err
		}
	}
	return CompileRuleset(spec.Rules), nil
}

func CompileRuleset(rules []SignatureRule) *CompiledRuleset {
	queryRules := make([]dfa.Rule, 0, len(rules))
	headerRules := make([]dfa.Rule, 0, len(rules))
	pathRules := make([]dfa.Rule, 0, len(rules))
	anyRules := make([]dfa.Rule, 0, len(rules))

	for i, rule := range rules {
		ruleID := strings.TrimSpace(rule.Name)
		if ruleID == "" {
			ruleID = "rule_" + strconvI(i)
		}
		tokens := extractPatternTokens(rule.Pattern)
		for _, token := range tokens {
			item := dfa.Rule{ID: ruleID, Weight: rule.Weight, Token: token}
			switch strings.ToLower(string(rule.Match)) {
			case string(MatchQuery):
				queryRules = append(queryRules, item)
			case string(MatchHeaders):
				headerRules = append(headerRules, item)
			case string(MatchPath):
				pathRules = append(pathRules, item)
			default:
				anyRules = append(anyRules, item)
			}
		}
	}

	return &CompiledRuleset{
		Any:     dfa.New(anyRules),
		Query:   dfa.New(queryRules),
		Headers: dfa.New(headerRules),
		Path:    dfa.New(pathRules),
	}
}

func (r *CompiledRuleset) Stats() RulesetStats {
	if r == nil {
		return RulesetStats{}
	}
	stats := RulesetStats{
		AnyRules:    r.Any.RuleCount(),
		QueryRules:  r.Query.RuleCount(),
		HeaderRules: r.Headers.RuleCount(),
		PathRules:   r.Path.RuleCount(),
		AnyNodes:    r.Any.NodeCount(),
		QueryNodes:  r.Query.NodeCount(),
		HeaderNodes: r.Headers.NodeCount(),
		PathNodes:   r.Path.NodeCount(),
	}
	stats.TotalRules = stats.AnyRules + stats.QueryRules + stats.HeaderRules + stats.PathRules
	stats.TotalNodes = stats.AnyNodes + stats.QueryNodes + stats.HeaderNodes + stats.PathNodes
	return stats
}

func extractPatternTokens(pattern string) []string {
	p := strings.TrimSpace(pattern)
	if p == "" {
		return nil
	}
	p = strings.TrimPrefix(p, "(?i)")
	p = strings.TrimPrefix(p, "(?I)")
	p = strings.TrimPrefix(p, "(")
	p = strings.TrimSuffix(p, ")")
	parts := strings.Split(p, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.ToLower(strings.TrimSpace(part))
		token = strings.Trim(token, "()")
		if token == "" {
			continue
		}
		out = append(out, token)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func strconvI(i int) string {
	if i == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	return string(buf[pos:])
}
