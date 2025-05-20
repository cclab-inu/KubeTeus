package policygenerator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PolicyGenerator(ctx context.Context, k8sClient client.Client, model string, prompts utils.Prompt) (utils.Policy, error) {
	var policy utils.Policy

	config, err := readConfig("conf/custom.yaml")
	if err != nil {
		return utils.Policy{}, err
	}
	token := config.User.HuggingfaceToken
	scriptPath := filepath.Join("pkg", "policygenerator", "models", "config-policy-generate-pipe.py")
	models := selectModels(model)

	var wg sync.WaitGroup
	errorCh := make(chan error, 2)
	defer close(errorCh)

	timeout := 90 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if len(prompts.Network) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			networkPolicies, err := ExecuteLLM(ctx, models.Network, prompts.Network, token, scriptPath)
			if err != nil {
				errorCh <- err
				return
			}
			policy.Network = networkPolicies
		}()
	}
	wg.Wait()

	select {
	case err := <-errorCh:
		return utils.Policy{}, err
	default:
	}
	return policy, nil
}

func selectModels(baseModel string) utils.Models {
	if baseModel == "default" {
		return utils.Models{
			Network: "cclabadmin/codegemma-7b-it-network",
		}
	}
	return utils.Models{
		Network: baseModel,
	}
}

func ExecuteLLM(ctx context.Context, model string, prompts []string, token string, scriptPath string) ([]string, error) {
	var results []string
	for _, prompt := range prompts {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("execution timed out")
		default:
			args := []string{scriptPath, "--model=" + model, "--prompt=" + prompt, "--token=" + token}
			cmd := exec.CommandContext(ctx, "python3", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("failed to execute script: %v, output: %s", err, string(output))
			}
			fmt.Printf("Script output: %s\n", string(output))

			if strings.HasPrefix(string(output), "Generated text:") {
				result := strings.TrimSpace(strings.TrimPrefix(string(output), "Generated text:"))
				results = append(results, result)
			} else {
				return nil, fmt.Errorf("desired output not found in script response")
			}
		}
	}
	return results, nil
}
