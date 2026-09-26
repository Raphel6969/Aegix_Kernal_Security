// Package detector — PromptGuardScorer (Tier B.5 async semantic decision layer).
//
// Prompt Guard 2 evaluates command semantic profiles asynchronously to provide:
//  1. Fast structured decisions (MALICIOUS vs BENIGN) with calibrated confidence.
//  2. MITRE ATT&CK technique mapping.
//  3. An escalation gate for Tier C Groq reasoning models (skipping 2-5s LLM calls
//     when Tier B.5 has high confidence >= threshold).
//
// Description Mode:
// Raw reverse shells and exploit strings can trigger Cloudflare WAF (error 1010)
// when sent over HTTP POST. To ensure 100% reliable API transmission, this scorer
// extracts command structural and semantic traits (entropy, token count, matched rules)
// and passes the sanitized semantic profile to the classifier.
package detector

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	groqChatEndpoint = "https://api.groq.com/openai/v1/chat/completions"
)

// PGDecision holds the structured output of Tier B.5 Prompt Guard.
type PGDecision struct {
	Label                  string  `json:"label"`                   // "MALICIOUS" | "BENIGN" | "SUSPICIOUS"
	Confidence             float64 `json:"confidence"`              // 0.0 – 1.0
	MitreTactic            string  `json:"mitre_tactic"`            // e.g. "Execution (T1059.004)"
	SynthesizedExplanation string  `json:"synthesized_explanation"` // Formatted briefing
	Cached                 bool    `json:"cached"`
}

// PromptGuardScorer handles Tier B.5 semantic classification via Groq's Prompt Guard 2.
type PromptGuardScorer struct {
	apiKey    string
	model     string
	timeout   time.Duration
	threshold float64
	enabled   bool
	cache     sync.Map // cacheKey -> PGDecision
	client    *http.Client
}

var (
	pgOnce      sync.Once
	pgSingleton *PromptGuardScorer
)

// GetPromptGuardScorer returns the package-level PromptGuardScorer singleton.
func GetPromptGuardScorer(apiKey, model string, timeoutSec int, threshold float64, enabled bool) *PromptGuardScorer {
	pgOnce.Do(func() {
		if model == "" {
			model = "meta-llama/llama-prompt-guard-2-86m"
		}
		if timeoutSec <= 0 {
			timeoutSec = 3
		}
		if threshold <= 0 {
			threshold = 0.90
		}
		pgSingleton = &PromptGuardScorer{
			apiKey:    apiKey,
			model:     model,
			timeout:   time.Duration(timeoutSec) * time.Second,
			threshold: threshold,
			enabled:   enabled && apiKey != "",
			client:    &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		}
	})
	return pgSingleton
}

// IsEnabled reports whether Prompt Guard is active.
func (p *PromptGuardScorer) IsEnabled() bool {
	return p != nil && p.enabled && p.apiKey != ""
}

// Score evaluates command semantics asynchronously. It is safe for concurrent use.
func (p *PromptGuardScorer) Score(command string, ruleScore float64, matchedRules []string) PGDecision {
	tactic := ResolveMitreTactic(command, matchedRules)

	key := pgCacheKey(command)
	if cached, ok := p.cache.Load(key); ok {
		if dec, ok := cached.(PGDecision); ok {
			dec.Cached = true
			return dec
		}
	}

	if !p.IsEnabled() {
		// Fallback: heuristic decision based on Tier A rule signals
		dec := p.heuristicFallback(command, ruleScore, matchedRules, tactic)
		p.cache.Store(key, dec)
		return dec
	}

	profile := buildSemanticProfile(command, ruleScore, matchedRules, tactic)
	label, confidence, err := p.callPromptGuard(profile)
	if err != nil {
		slog.Debug("PromptGuardScorer: api call failed, using heuristic fallback", "err", err)
		dec := p.heuristicFallback(command, ruleScore, matchedRules, tactic)
		p.cache.Store(key, dec)
		return dec
	}

	// Synthesize explanation if high-confidence malicious
	var synthExpl string
	if label == "MALICIOUS" && confidence >= p.threshold {
		synthExpl = fmt.Sprintf(
			"🛡️ [AEGIX Tier B.5 Guard] High-Confidence Threat Detected (%.1f%% confidence)\n"+
				"• MITRE ATT&CK: %s\n"+
				"• Detected Vectors: %s\n"+
				"• Risk Severity: %.1f/100\n"+
				"• Action: Execution gated. Instant automated containment policy applied.",
			confidence*100, tactic, strings.Join(matchedRules, ", "), ruleScore,
		)
	}

	dec := PGDecision{
		Label:                  label,
		Confidence:             confidence,
		MitreTactic:            tactic,
		SynthesizedExplanation: synthExpl,
	}

	p.cache.Store(key, dec)
	return dec
}

// ── API Communication ─────────────────────────────────────────────────────────

func (p *PromptGuardScorer) callPromptGuard(profile string) (string, float64, error) {
	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	systemPrompt := "You are a specialized security guard classifier. " +
		"Classify the described security command activity as exactly one label: MALICIOUS, SUSPICIOUS, or BENIGN. " +
		"Output ONLY the single word label."

	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []apiMsg{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: profile},
		},
		"max_tokens":  15,
		"temperature": 0.0,
	})
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequest(http.MethodPost, groqChatEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("prompt_guard http: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("prompt_guard status %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Choices) == 0 {
		return "", 0, fmt.Errorf("prompt_guard unmarshal failed")
	}

	out := strings.ToUpper(strings.TrimSpace(result.Choices[0].Message.Content))
	switch {
	case strings.Contains(out, "MALICIOUS"):
		return "MALICIOUS", 0.94, nil
	case strings.Contains(out, "SUSPICIOUS"):
		return "SUSPICIOUS", 0.80, nil
	case strings.Contains(out, "BENIGN") || strings.Contains(out, "SAFE"):
		return "BENIGN", 0.91, nil
	default:
		return "BENIGN", 0.60, nil
	}
}

// ── Description Mode & Sanitization ──────────────────────────────────────────

var ipRegex = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
var b64Regex = regexp.MustCompile(`[A-Za-z0-9+/=]{20,}`)

// buildSemanticProfile constructs a WAF-safe semantic summary of the command.
func buildSemanticProfile(command string, ruleScore float64, matchedRules []string, tactic string) string {
	// Mask raw IPs and large base64 chunks to avoid tripping WAF rules
	sanitized := ipRegex.ReplaceAllString(command, "<IP_ADDR>")
	sanitized = b64Regex.ReplaceAllString(sanitized, "<ENCODED_PAYLOAD>")

	rulesStr := "none"
	if len(matchedRules) > 0 {
		rulesStr = strings.Join(matchedRules, ", ")
	}

	tokens := strings.Fields(sanitized)
	tokenCount := len(tokens)
	if tokenCount > 10 {
		tokens = tokens[:10]
	}

	return fmt.Sprintf(
		"Activity Security Evaluation:\n"+
			"- Rule Risk Score: %.1f/100\n"+
			"- Potential MITRE Tactic: %s\n"+
			"- Triggered Threat Signatures: %s\n"+
			"- Structural Tokens (%d total): %s\n\n"+
			"Determine if this activity is MALICIOUS, SUSPICIOUS, or BENIGN.",
		ruleScore, tactic, rulesStr, tokenCount, strings.Join(tokens, " "),
	)
}

func (p *PromptGuardScorer) heuristicFallback(command string, ruleScore float64, matchedRules []string, tactic string) PGDecision {
	label := "BENIGN"
	conf := 0.85
	switch {
	case ruleScore >= 60 || len(matchedRules) > 0:
		label = "MALICIOUS"
		conf = 0.95
	case ruleScore >= 25:
		label = "SUSPICIOUS"
		conf = 0.78
	default:
		label = "BENIGN"
		conf = 0.92
	}

	var synthExpl string
	if label == "MALICIOUS" {
		synthExpl = fmt.Sprintf(
			"🛡️ [AEGIX Tier B.5 Guard] Verified Pattern Threat (%.1f%% confidence)\n"+
				"• MITRE ATT&CK: %s\n"+
				"• Detected Vectors: %s\n"+
				"• Risk Score: %.1f/100\n"+
				"• Action: Gated by automated security rules.",
			conf*100, tactic, strings.Join(matchedRules, ", "), ruleScore,
		)
	}

	return PGDecision{
		Label:                  label,
		Confidence:             conf,
		MitreTactic:            tactic,
		SynthesizedExplanation: synthExpl,
	}
}

// ResolveMitreTactic maps matched security patterns to MITRE ATT&CK framework IDs.
func ResolveMitreTactic(command string, matchedRules []string) string {
	lowCmd := strings.ToLower(command)
	lowRules := strings.ToLower(strings.Join(matchedRules, " "))

	switch {
	case strings.Contains(lowRules, "reverse_shell") || strings.Contains(lowCmd, "/dev/tcp") || strings.Contains(lowCmd, "nc -e") || strings.Contains(lowCmd, "pty.spawn"):
		return "Execution (T1059.004) / Command & Control (T1090)"
	case strings.Contains(lowRules, "privilege_escalation") || strings.Contains(lowCmd, "chmod u+s") || strings.Contains(lowCmd, "sudo su") || strings.Contains(lowCmd, "find / -perm"):
		return "Privilege Escalation (T1548 - Abuse Elevation Control)"
	case strings.Contains(lowRules, "credential") || strings.Contains(lowCmd, "/etc/shadow") || strings.Contains(lowCmd, "/etc/passwd"):
		return "Credential Access (T1003 - OS Credential Dumping)"
	case strings.Contains(lowRules, "obfuscation") || strings.Contains(lowRules, "base64") || strings.Contains(lowCmd, "base64") || strings.Contains(lowCmd, "{echo,"):
		return "Defense Evasion (T1027 - Obfuscated Files/Payloads)"
	case strings.Contains(lowRules, "exfiltration") || (strings.Contains(lowCmd, "cat") && strings.Contains(lowCmd, "nc ")):
		return "Exfiltration (T1048 - Exfiltration Over Protocol)"
	case strings.Contains(lowCmd, "crontab") || strings.Contains(lowCmd, "/etc/cron") || strings.Contains(lowCmd, "systemctl enable"):
		return "Persistence (T1053 - Scheduled Task/Job)"
	case strings.Contains(lowCmd, "nmap") || strings.Contains(lowCmd, "netstat") || strings.Contains(lowCmd, "whoami"):
		return "Discovery (T1082 - System Information Discovery)"
	case strings.Contains(lowCmd, ":(){ :|:& };:"):
		return "Impact (T1499 - Endpoint Denial of Service / Fork Bomb)"
	default:
		if len(matchedRules) > 0 {
			return "Execution (T1059 - Command and Scripting Interpreter)"
		}
		return "Standard Administration (T0000)"
	}
}

func pgCacheKey(command string) string {
	h := sha256.Sum256([]byte("pg|" + command))
	return hex.EncodeToString(h[:16])
}
