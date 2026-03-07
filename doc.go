// Package guardgo provides a Redis-backed API protection middleware with a single
// atomic Redis roundtrip (Lua) on the critical path and optional asynchronous
// pattern analysis (rules) off the critical path.
package guardgo
