package guardgo

import "net/http"

// Rule evaluates a request and returns an integer threat weight.
// Higher values represent higher confidence/more dangerous patterns.
//
// Rules MUST NOT read or depend on r.Body. They may read URL, Method, Host and headers.
type Rule interface {
	Evaluate(r *http.Request) int
}
