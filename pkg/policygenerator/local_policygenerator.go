package policygenerator

/*
func PolicyGenerator(ctx context.Context, k8sClient client.Client, model string, prompts utils.Prompt) (utils.Policy, error) {
	var policy utils.Policy
	const maxRetries = 5

	config, err := readConfig("conf/custom.yaml")
	if err != nil {
		return utils.Policy{}, err
	}

	token := config.User.HuggingfaceToken
	scriptPath := filepath.Join("pkg", "policygenerator", "models", "config-policy-generate.py")

	models := selectModels(model)

	if len(prompts.Network) > 0 {
		for attempt := 1; attempt <= maxRetries; attempt++ {
			networkPolicies, err := ExecuteLLM(models.Network, prompts.Network, token, scriptPath)
			if err == nil {
				policy.Network = networkPolicies
				break
			}
			logFailureWithTimestamp(fmt.Sprintf("Network policy generation failed, attempt %d/%d: %v\n", attempt, maxRetries, err))
			time.Sleep(2 * time.Second)
		}
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

func ExecuteLLM(model string, prompts []string, token string, scriptPath string) ([]string, error) {
	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error

	for _, prompt := range prompts {
		wg.Add(1)
		go func(prompt string) {
			defer wg.Done()
			args := []string{scriptPath, "--model=" + model, "--prompt=" + prompt, "--token=" + token}
			cmd := exec.Command("python3", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to execute script: %v, output: %s", err, string(output)))
				saveOutputToFile("policygenerator.txt", string(output))
				mu.Unlock()
				return
			}

			filteredOutput := filterOutput(string(output))
			re := regexp.MustCompile(`(?s)\nkind:.*`)
			match := re.FindStringSubmatch(filteredOutput)
			if len(match) > 0 {
				mu.Lock()
				results = append(results, match[0])
				mu.Unlock()
			} else {
				mu.Lock()
				errs = append(errs, fmt.Errorf("desired output not found in script response: %s", filteredOutput))
				mu.Unlock()
			}
		}(prompt)
	}
	wg.Wait()
	if len(errs) > 0 {
		return nil, errs[0]
	}
	return results, nil
}
*/
