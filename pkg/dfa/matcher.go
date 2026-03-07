package dfa

import "strings"

type Rule struct {
	ID     string
	Weight int64
	Token  string
}

type Match struct {
	RuleID string
	Weight int64
}

type Matcher struct {
	nodes []node
	rules []Rule
}

type node struct {
	next [256]int
	fail int
	out  []int
}

func New(rules []Rule) *Matcher {
	m := &Matcher{
		nodes: []node{newNode()},
		rules: make([]Rule, 0, len(rules)),
	}
	for _, rule := range rules {
		token := strings.TrimSpace(strings.ToLower(rule.Token))
		if token == "" {
			continue
		}
		if rule.Weight <= 0 {
			rule.Weight = 1
		}
		rule.Token = token
		m.rules = append(m.rules, rule)
		m.insert(token, len(m.rules)-1)
	}
	m.buildFailures()
	return m
}

func newNode() node {
	n := node{}
	for i := range n.next {
		n.next[i] = -1
	}
	return n
}

func (m *Matcher) insert(token string, ruleIndex int) {
	state := 0
	for i := 0; i < len(token); i++ {
		b := token[i]
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		next := m.nodes[state].next[b]
		if next == -1 {
			m.nodes = append(m.nodes, newNode())
			next = len(m.nodes) - 1
			m.nodes[state].next[b] = next
		}
		state = next
	}
	m.nodes[state].out = append(m.nodes[state].out, ruleIndex)
}

func (m *Matcher) buildFailures() {
	queue := make([]int, 0, len(m.nodes))

	for b := 0; b < 256; b++ {
		next := m.nodes[0].next[b]
		if next == -1 {
			m.nodes[0].next[b] = 0
			continue
		}
		m.nodes[next].fail = 0
		queue = append(queue, next)
	}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		for b := 0; b < 256; b++ {
			next := m.nodes[state].next[b]
			if next == -1 {
				m.nodes[state].next[b] = m.nodes[m.nodes[state].fail].next[b]
				continue
			}
			fail := m.nodes[state].fail
			m.nodes[next].fail = m.nodes[fail].next[b]
			m.nodes[next].out = append(m.nodes[next].out, m.nodes[m.nodes[next].fail].out...)
			queue = append(queue, next)
		}
	}
}

func (m *Matcher) FindAll(text string) []Match {
	if len(m.rules) == 0 || text == "" {
		return nil
	}

	seen := make(map[int]struct{}, 8)
	state := 0
	for i := 0; i < len(text); i++ {
		b := text[i]
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		state = m.nodes[state].next[b]
		for _, ruleIndex := range m.nodes[state].out {
			seen[ruleIndex] = struct{}{}
		}
	}

	matches := make([]Match, 0, len(seen))
	for ruleIndex := range seen {
		r := m.rules[ruleIndex]
		matches = append(matches, Match{
			RuleID: r.ID,
			Weight: r.Weight,
		})
	}
	return matches
}

func (m *Matcher) RuleCount() int {
	if m == nil {
		return 0
	}
	return len(m.rules)
}

func (m *Matcher) NodeCount() int {
	if m == nil {
		return 0
	}
	return len(m.nodes)
}
