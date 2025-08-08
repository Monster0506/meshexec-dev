package executor

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

type SafetyChecker struct {
	cfg       *internal.Config
	logger    *logging.Logger
	patterns  []*regexp.Regexp
	maxLength int
	safeMode  bool
}

func NewSafetyChecker(cfg *internal.Config, logger *logging.Logger) *SafetyChecker {
	checker := &SafetyChecker{
		cfg:       cfg,
		logger:    logger,
		maxLength: max(1, cfg.Safety.MaxCommandLength),
		safeMode:  cfg.Safety.SafeMode,
	}
	checker.compilePatterns()
	return checker
}

func (s *SafetyChecker) compilePatterns() {
	var raw []string
	// defaults by platform
	if runtime.GOOS == "windows" {
		raw = append(raw,
			`(?:(?:^|\s))(?:del)\s+/s(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:rd)\s+/s(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:format)(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:bcdedit)(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:shutdown)(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:cipher)\s+/w(?:(?:\s|$))`,
		)
		// PowerShell cmdlets
		raw = append(raw,
			`powershell\b.*-command\b.*remove-item.*-recurse`,
			`remove-item\b.*-recurse`,
			`stop-computer\b`,
			`format-volume\b`,
		)
	} else {
		raw = append(raw,
			`(?:(?:^|\s))(?:rm)\s+-rf(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:dd)\s+if=[^-\s]*(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:shutdown)(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:poweroff)(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:mkfs)(?:(?:\s|$))`,
			`(?:(?:^|\s))(?:chmod)\s+-R\s+000\s+/(?:(?:\s|$))`,
			// shell wrapper
			`(?:(?:^|\s))(?:sh|bash)\s+-c\s+.*rm\s+-rf(?:(?:\s|$))`,
		)
		// fork bomb pattern (loose)
		raw = append(raw, `:\(\)\s*\{\s*:\|:\s*&\s*\};:`)
	}
	// merge config
	for _, p := range s.cfg.Safety.DangerousCommands {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// build token-anchored flexible whitespace pattern without over-escaping
		tokens := strings.Fields(p)
		if len(tokens) == 0 {
			continue
		}
		var b strings.Builder
		b.WriteString(`(?:(?:^|\s))`)
		for i, tok := range tokens {
			b.WriteString(regexp.QuoteMeta(strings.ToLower(tok)))
			if i < len(tokens)-1 {
				b.WriteString(`\s+`)
			}
		}
		b.WriteString(`(?:(?:\s|$))`)
		raw = append(raw, b.String())
	}
	// compile
	for _, rp := range raw {
		re, err := regexp.Compile(`(?i)` + rp)
		if err == nil {
			s.patterns = append(s.patterns, re)
		}
	}
}

func (s *SafetyChecker) ValidateCommand(cmd string) error {
	if !s.safeMode {
		return nil
	}
	if s.logger != nil {
		s.logger.Debug("Validating command safety", map[string]interface{}{"len": len(cmd)})
	}
	// length
	if len(cmd) > s.maxLength {
		return &internal.MeshExecError{Type: internal.ErrorTypeExecution, Code: "command_too_long", Message: fmt.Sprintf("command exceeds max length (%d)", s.maxLength), Details: map[string]interface{}{"max": s.maxLength, "len": len(cmd)}}
	}
	// normalize for matching
	norm := normalize(cmd)
	for _, re := range s.patterns {
		if re.MatchString(norm) {
			if s.logger != nil {
				s.logger.Warn("Blocked dangerous command", map[string]interface{}{"pattern": re.String()})
			}
			return &internal.MeshExecError{Type: internal.ErrorTypeExecution, Code: "dangerous_command", Message: "command blocked by safety policy", Details: map[string]interface{}{"pattern": re.String()}}
		}
	}
	return nil
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	// collapse whitespace to single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
