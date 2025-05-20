package policygenerator

/*

func PolicyGenerator(ctx context.Context, k8sClient client.Client, model string, prompts utils.Prompt) (utils.Policy, error) {
	var policy utils.Policy
	logger := log.FromContext(ctx)
	models := utils.SelectModel(model)

	if len(prompts.Network) > 0 {
		networkPolicies, err := generatePolicy(ctx, prompts.Network, "http://localhost:5000/generate", models.Network)
		if err == nil {
			policy.Network = networkPolicies
		} else {
			logger.Error(err, "Network policy generation failed")
		}
	}

	return policy, nil
}

func generatePolicy(ctx context.Context, prompts []string, url string, model string) ([]string, error) {
	logger := log.FromContext(ctx)
	var results []string
	var mu sync.Mutex

	for _, prompt := range prompts {
		cmd := exec.Command("curl", "-X", "POST", url, "-H", "Content-Type: application/json", "-d", fmt.Sprintf(`{"model": "%s", "prompt": "%s"}`, model, prompt))

		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			return nil, fmt.Errorf("failed to execute command: %v", err)
		}

		if cmd.ProcessState.ExitCode() == 0 {
			logger.Info("Policy generation succeeded", "Output", out.String())

			output := out.String()
			re := regexp.MustCompile(`(?s)\nkind:.*`)
			match := re.FindStringSubmatch(output)
			if len(match) > 0 {
				mu.Lock()
				results = append(results, match[0])
				mu.Unlock()
			} else {
				return nil, fmt.Errorf("desired output not found in script response")
			}
		} else {
			logger.Error(fmt.Errorf("non-200 response"), "Policy generation failed with non-200 response.")
			continue
		}
	}

	logger.Info("Policy generated", "Result", results)
	return results, nil
}

*/
