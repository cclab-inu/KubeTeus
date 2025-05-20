package promptprocessor

import (
	"context"
	"fmt"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenerateCiliumPrompts(ctx context.Context, k8sClient client.Client, netInfo *utils.NetInfo) ([]string, error) {
	podList, err := listPodsByNamespace(ctx, k8sClient, netInfo.Namespace)
	if err != nil {
		return nil, err
	}

	var prompts []string

	for _, pod := range podList {
		if pod.Name == netInfo.Name {
			continue // Skip the target pod itself
		}

		otherNetInfo, err := getPodNetInfo(ctx, k8sClient, pod.Name, pod.Namespace)
		if err != nil {
			return nil, err
		}

		// Check if this pod refers to the target pod in its environment variables
		for _, env := range otherNetInfo.RelatedEnv {
			if env.Name == netInfo.ServiceName {
				labelSelectors := formatLabels(netInfo.Labels)
				prompt := fmt.Sprintf("Create a CiliumNetworkPolicy that allows incoming ingress traffic over '%s/%s' from '%s' Pod to the '%s pod' labeled '%s'", netInfo.ContainerPorts, netInfo.Protocol, otherNetInfo.Name, netInfo.Name, labelSelectors)
				prompts = append(prompts, prompt)
				break
			}
		}
	}

	return prompts, nil
}
