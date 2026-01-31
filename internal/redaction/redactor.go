package redaction

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kadirbelkuyu/kubecrsh/internal/config"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type compiledRule struct {
	re   *regexp.Regexp
	repl string
}

type Redactor struct {
	replacement      string
	envAllowlist     []string
	envDenylist      []string
	logRules         []compiledRule
	redactFromSource bool
}

func New(cfg config.RedactionConfig) (*Redactor, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	replacement := strings.TrimSpace(cfg.Replacement)
	if replacement == "" {
		replacement = "***"
	}

	patterns := cfg.LogPatterns
	if len(patterns) == 0 {
		patterns = defaultLogPatterns()
	}

	rules := make([]compiledRule, 0, len(patterns))
	for _, raw := range patterns {
		pat, repl := splitRule(raw, replacement)
		if strings.TrimSpace(pat) == "" {
			continue
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, fmt.Errorf("invalid log redaction pattern: %w", err)
		}
		rules = append(rules, compiledRule{re: re, repl: repl})
	}

	return &Redactor{
		replacement:      replacement,
		envAllowlist:     cfg.EnvAllowlist,
		envDenylist:      cfg.EnvDenylist,
		logRules:         rules,
		redactFromSource: cfg.RedactFromSource,
	}, nil
}

func (r *Redactor) Apply(report *domain.ForensicReport) {
	if report == nil {
		return
	}

	if report.EnvVars != nil {
		for k, v := range report.EnvVars {
			if !r.redactFromSource && v == "[from-source]" {
				continue
			}

			if len(r.envAllowlist) > 0 && !matchAny(r.envAllowlist, k) {
				report.EnvVars[k] = r.replacement
				continue
			}

			if len(r.envAllowlist) == 0 && len(r.envDenylist) == 0 {
				report.EnvVars[k] = r.replacement
				continue
			}

			if len(r.envDenylist) > 0 && matchAny(r.envDenylist, k) {
				report.EnvVars[k] = r.replacement
			}
		}
	}

	report.Logs = r.redactLines(report.Logs)
	report.PreviousLog = r.redactLines(report.PreviousLog)
}

func (r *Redactor) redactLines(lines []string) []string {
	if len(lines) == 0 || len(r.logRules) == 0 {
		return lines
	}

	out := make([]string, len(lines))
	for i, line := range lines {
		s := line
		for _, rule := range r.logRules {
			s = rule.re.ReplaceAllString(s, rule.repl)
		}
		out[i] = s
	}
	return out
}

func matchAny(patterns []string, s string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == s {
			return true
		}
		if ok, _ := filepath.Match(p, s); ok {
			return true
		}
	}
	return false
}

func splitRule(raw, fallback string) (string, string) {
	raw = strings.TrimSpace(raw)
	parts := strings.SplitN(raw, "=>", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return raw, fallback
}

func defaultLogPatterns() []string {
	return []string{
		`(?i)((?:authorization|x-authorization)\s*:\s*bearer\s+)[^\s]+=>$1***`,
		`(?i)((?:token|api[_-]?key|secret|password)\s*[:=]\s*)[^\s]+=>$1***`,
		`(?i)((?:client[_-]?secret)\s*[:=]\s*)[^\s]+=>$1***`,
	}
}
