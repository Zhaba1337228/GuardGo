package rules

import "guardgo"

// DefaultSecurityPresets returns a production-oriented baseline signatures set.
func DefaultSecurityPresets() []guardgo.SignatureRule {
	return []guardgo.SignatureRule{
		{
			Name:    "SQLi Injection",
			Match:   guardgo.MatchQuery,
			Pattern: "(?i)(union|select|drop|insert|or 1=1|benchmark\\()",
			Weight:  15,
		},
		{
			Name:    "Path Traversal",
			Match:   guardgo.MatchPath,
			Pattern: "(?i)(\\.\\./|/etc/passwd|/proc/self/environ)",
			Weight:  20,
		},
		{
			Name:    "Sensitive Files",
			Match:   guardgo.MatchPath,
			Pattern: "(?i)(\\.env|id_rsa|config\\.php|wp-config\\.php)",
			Weight:  18,
		},
		{
			Name:    "Common Scanners",
			Match:   guardgo.MatchHeaders,
			Pattern: "(?i)(sqlmap|nmap|nikto|acunetix|masscan|zgrab)",
			Weight:  25,
		},
		{
			Name:    "Command Injection",
			Match:   guardgo.MatchQuery,
			Pattern: "(?i)(;\\s*cat\\s+|\\|\\s*sh\\s|`\\s*wget\\s)",
			Weight:  20,
		},
	}
}
