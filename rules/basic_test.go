package rules

import (
	"net/http"
	"testing"
)

func TestPathContainsEvaluate(t *testing.T) {
	testCases := []struct {
		name       string
		path       string
		tokens     []string
		weight     int
		wantWeight int
	}{
		{
			name:       "token_match_case_insensitive",
			path:       "/Admin/Login",
			tokens:     []string{"admin"},
			weight:     3,
			wantWeight: 3,
		},
		{
			name:       "no_match_returns_zero",
			path:       "/public",
			tokens:     []string{"admin"},
			weight:     3,
			wantWeight: 0,
		},
		{
			name:       "default_weight_is_one",
			path:       "/wp-admin",
			tokens:     []string{"wp-admin"},
			weight:     0,
			wantWeight: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "http://example.com"+testCase.path, nil)
			rule := PathContains{
				Tokens: testCase.tokens,
				Weight: testCase.weight,
			}
			got := rule.Evaluate(req)
			if got != testCase.wantWeight {
				t.Fatalf("unexpected weight: got=%d want=%d", got, testCase.wantWeight)
			}
		})
	}
}

func TestQueryContainsEvaluate(t *testing.T) {
	testCases := []struct {
		name       string
		query      string
		tokens     []string
		weight     int
		wantWeight int
	}{
		{
			name:       "token_match_case_insensitive",
			query:      "q=UNION+SELECT+1",
			tokens:     []string{"union+select"},
			weight:     5,
			wantWeight: 5,
		},
		{
			name:       "no_match_returns_zero",
			query:      "q=safe",
			tokens:     []string{"union+select"},
			weight:     5,
			wantWeight: 0,
		},
		{
			name:       "default_weight_is_one",
			query:      "x=or+1=1",
			tokens:     []string{"or+1=1"},
			weight:     -1,
			wantWeight: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "http://example.com/test?"+testCase.query, nil)
			rule := QueryContains{
				Tokens: testCase.tokens,
				Weight: testCase.weight,
			}
			got := rule.Evaluate(req)
			if got != testCase.wantWeight {
				t.Fatalf("unexpected weight: got=%d want=%d", got, testCase.wantWeight)
			}
		})
	}
}

func TestHeaderContainsEvaluate(t *testing.T) {
	testCases := []struct {
		name       string
		headerName string
		headerVal  string
		tokens     []string
		weight     int
		wantWeight int
	}{
		{
			name:       "token_match_case_insensitive",
			headerName: "User-Agent",
			headerVal:  "sqlmap/1.7",
			tokens:     []string{"SQLMAP"},
			weight:     7,
			wantWeight: 7,
		},
		{
			name:       "no_match_returns_zero",
			headerName: "User-Agent",
			headerVal:  "mozilla/5.0",
			tokens:     []string{"nikto"},
			weight:     7,
			wantWeight: 0,
		},
		{
			name:       "default_weight_is_one",
			headerName: "X-Test",
			headerVal:  "contains-bad-token",
			tokens:     []string{"bad-token"},
			weight:     0,
			wantWeight: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
			req.Header.Set(testCase.headerName, testCase.headerVal)
			rule := HeaderContains{
				Name:   testCase.headerName,
				Tokens: testCase.tokens,
				Weight: testCase.weight,
			}
			got := rule.Evaluate(req)
			if got != testCase.wantWeight {
				t.Fatalf("unexpected weight: got=%d want=%d", got, testCase.wantWeight)
			}
		})
	}
}
