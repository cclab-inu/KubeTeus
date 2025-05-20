package policyconverter

import (
	"context"
	"regexp"
	"strings"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	PATH_PATTERN    = regexp.MustCompile(`(/[a-zA-Z_][-a-zA-Z0-9_/.]*[^/])`)
	FILE_PATTERN    = regexp.MustCompile(`\b([a-zA-Z_][-a-zA-Z0-9_/.]*\.(conf|txt|bin|log|cfg))\b`)
	KEY_VAL_PATTERN = regexp.MustCompile(`\s*(\w+)\s*:\s*(\S+)\s*`)
)

func PolicyProcessor(ctx context.Context, podInfo utils.NetInfo, entities []utils.Entity, policies utils.Policy) (interface{}, []error) {
	logger := log.FromContext(ctx)
	// logger.Info("Policy received", "Pod.Info", podInfo, "Policy", policies)

	processedNetworkPolicies, networkErrors := processNetworkPolicies(ctx, podInfo, entities, policies.Network)

	var allPolicies []interface{}
	allPolicies = append(allPolicies, processedNetworkPolicies...)

	var allErrors []error
	allErrors = append(allErrors, networkErrors...)

	logger.Info("Policies processed", "Policy.Details", allPolicies)
	return allPolicies, allErrors
}

func cleanPolicyString(policy string) string {
	lines := strings.Split(policy, "\n")
	var cleanLines []string
	seenLines := make(map[string]bool)
	actionPatterns := []string{
		"action: Block", "action: Allow", "action: Audit",
		"action:\n Block", "action:\n Allow", "action:\n Audit",
	}

	for _, line := range lines {
		cleanLine := strings.TrimSpace(line)
		if cleanLine != "" && !strings.HasPrefix(cleanLine, "➜") && !strings.HasPrefix(cleanLine, "©") && !strings.HasPrefix(cleanLine, "✗") && !strings.HasPrefix(cleanLine, "~ kubectl") && !strings.HasPrefix(cleanLine, "NAME") && !strings.HasPrefix(cleanLine, "AGE") && !strings.HasPrefix(cleanLine, "READY") && !strings.HasPrefix(cleanLine, "STATUS") && !strings.HasPrefix(cleanLine, "RESTARTS") && !strings.HasPrefix(cleanLine, "IP") && !strings.HasPrefix(cleanLine, "NODE") {
			if !seenLines[cleanLine] {
				cleanLines = append(cleanLines, cleanLine)
				seenLines[cleanLine] = true
			}
		}
		for _, pattern := range actionPatterns {
			if strings.Contains(cleanLine, pattern) {
				return strings.Join(cleanLines, "\n")
			}
		}
	}

	return strings.Join(cleanLines, "\n")
}

func getPathForName(entities []utils.Entity) string {
	for _, entity := range entities {
		if entity.Type == "PATH" {
			cleanPath := strings.ReplaceAll(strings.ReplaceAll(strings.Trim(entity.Text, "/"), "/", ""), ".", "")
			if FILE_PATTERN.MatchString(entity.Text) {
				return strings.SplitN(cleanPath, ".", 2)[0]
			}
			return cleanPath
		}
	}
	return ""
}
