package promptprocessor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func UserPromptProcessor(ctx context.Context, k8sClient client.Client, intent string, entities []utils.Entity) ([]string, utils.NetInfo, utils.NetInfo, error) {
	var netInfo, otherInfo utils.NetInfo
	var finalIntents []string

	cniType, err := detectCNIPlugin(ctx, k8sClient)
	if err != nil {
		return nil, utils.NetInfo{}, utils.NetInfo{}, fmt.Errorf("failed to detect CNI: %v", err)
	}

	finalIntent, label, netInfo, _, podNameExists, err := processCommonEntities(ctx, k8sClient, intent, entities)
	if err != nil {
		return nil, utils.NetInfo{}, utils.NetInfo{}, err
	}

	trafficDirection := getTrafficDirection(entities)
	if podNameExists && containsAnotherPod(entities) {
		finalIntents, netInfo, otherInfo, err = processTwoPods(ctx, k8sClient, finalIntent, label, netInfo, entities, trafficDirection)
	} else {
		finalIntents, netInfo, otherInfo, err = processSinglePodWithConfig(ctx, k8sClient, finalIntent, label, netInfo, entities, trafficDirection)
	}
	if err != nil {
		return nil, utils.NetInfo{}, utils.NetInfo{}, err
	}

	for i, intent := range finalIntents {
		for _, entity := range entities {
			if entity.Type == "POLICY" {
				switch cniType {
				case "cilium":
					finalIntents[i] = strings.ReplaceAll(intent, entity.Text, "CiliumNetworkPolicy")
				case "calico":
					finalIntents[i] = strings.ReplaceAll(intent, entity.Text, "CalicoNetworkPolicy")
				case "antrea":
					finalIntents[i] = strings.ReplaceAll(intent, entity.Text, "AntreaNetworkPolicy")
				case "weave":
					finalIntents[i] = strings.ReplaceAll(intent, entity.Text, "KubernetesNetworkPolicy")
				}
			}
		}
	}

	return finalIntents, netInfo, otherInfo, nil
}

func detectCNIPlugin(ctx context.Context, k8sClient client.Client) (string, error) {
	podList := &v1.PodList{}
	if err := k8sClient.List(ctx, podList, client.InNamespace("kube-system")); err != nil {
		return "", err
	}

	for _, pod := range podList.Items {
		name := strings.ToLower(pod.Name)
		switch {
		case strings.Contains(name, "cilium"):
			return "cilium", nil
		case strings.Contains(name, "calico"):
			return "calico", nil
		case strings.Contains(name, "antrea"):
			return "antrea", nil
		case strings.Contains(name, "weave"):
			return "weave", nil
		}
	}
	return "unknown", nil
}

func processCommonEntities(ctx context.Context, k8sClient client.Client, intent string, entities []utils.Entity) (string, string, utils.NetInfo, bool, bool, error) {
	logger := log.FromContext(ctx)
	var labelExists, podNameExists, namespaceExists bool
	var label, podName, namespace string
	var netInfo utils.NetInfo

	for _, entity := range entities {
		switch entity.Type {
		case "LABEL":
			labelExists = true
			label = entity.Text
		case "POD_NAME":
			podNameExists = true
			podName = entity.Text
		case "NAMESPACE":
			namespaceExists = true
			namespace = entity.Text
		}
	}

	if !namespaceExists {
		namespace = "default"
	}

	if podNameExists {
		if strings.HasSuffix(podName, " pod") {
			podName = strings.TrimSuffix(podName, " pod")
		}

		if podName == "pod" {
			logger.Info("No valid POD_NAME entity found in the intent")
			return "", "", utils.NetInfo{}, false, false, fmt.Errorf("invalid POD_NAME entity: 'pod'")
		}

		if strings.Count(intent, podName) == 2 {
			intent = strings.Replace(intent, podName+" pod", podName, 1)
			intent = strings.Replace(intent, podName, podName+" pod", 1)
		}

		podList, err := listPodsByNamespace(ctx, k8sClient, namespace)
		if err != nil {
			logger.Error(err, "Failed to list pods in namespace", "namespace", namespace)
			return "", "", utils.NetInfo{}, false, false, err
		}
		matchingPods := filterPodsByName(podList, podName)
		if len(matchingPods) == 1 {
			labelKey, labelValue := getFirstLabel(&matchingPods[0])
			label = fmt.Sprintf("%s: %s", labelKey, labelValue)
			intent = strings.ReplaceAll(intent, "pod", fmt.Sprintf("pod labeled '%s'", label))
		}

		// Fetch pod configuration using the GetConfig function from the watcher package
		netInfo, err = getPodNetInfo(ctx, k8sClient, podName, namespace)
		if err != nil {
			return "", "", utils.NetInfo{}, false, false, err
		}
	}
	return intent, label, netInfo, labelExists, podNameExists, nil
}

func processTwoPods(ctx context.Context, k8sClient client.Client, intent string, label string, netInfo utils.NetInfo, entities []utils.Entity, trafficDirection string) ([]string, utils.NetInfo, utils.NetInfo, error) {
	logger := log.FromContext(ctx)
	var otherInfo utils.NetInfo
	var finalIntents []string

	for _, entity := range entities {
		if entity.Type == "POD_NAME" && entity.Text != netInfo.Name {
			anotherPodName := entity.Text
			if strings.HasSuffix(anotherPodName, " pod") {
				anotherPodName = strings.TrimSuffix(anotherPodName, " pod")
			}
			anotherPodName = extractPodName(anotherPodName)

			otherNetInfo, err := getPodNetInfo(ctx, k8sClient, anotherPodName, netInfo.Namespace)
			if err != nil {
				logger.Error(err, "Failed to get pod net info", "podName", anotherPodName, "namespace", netInfo.Namespace)
				return nil, utils.NetInfo{}, utils.NetInfo{}, err
			}

			if otherNetInfo.Name != "" {
				otherInfo = otherNetInfo

				// 💡 기존 "from endpoint" 또는 "to endpoint" 제거
				intent = strings.ReplaceAll(intent, "from endpoint", "")
				intent = strings.ReplaceAll(intent, "to endpoint", "")
				intent = strings.TrimSpace(intent)

				if trafficDirection == "ingress" {
					finalIntent := strings.Replace(intent, "traffic", fmt.Sprintf("traffic from '%s pod' labeled '%s'", otherInfo.Name, otherInfo.Labels["app"]), 1)
					finalIntents = append(finalIntents, finalIntent)
				} else {
					finalIntent := strings.Replace(intent, "traffic", fmt.Sprintf("traffic to '%s pod' labeled '%s'", otherInfo.Name, otherInfo.Labels["app"]), 1)
					finalIntents = append(finalIntents, finalIntent)
				}
			}
		}
	}

	return finalIntents, netInfo, otherInfo, nil
}

func processSinglePodWithConfig(ctx context.Context, k8sClient client.Client, intent string, label string, netInfo utils.NetInfo, entities []utils.Entity, trafficDirection string) ([]string, utils.NetInfo, utils.NetInfo, error) {
	logger := log.FromContext(ctx)
	var otherInfo utils.NetInfo
	var finalIntents []string

	// Fetch the service's label key and value
	serviceLabelKey, serviceLabelValue, err := getServiceLabel(ctx, k8sClient, netInfo.Namespace, netInfo.ServiceName)
	if err != nil {
		logger.Error(err, "Failed to get service label", "serviceName", netInfo.ServiceName)
		return nil, utils.NetInfo{}, utils.NetInfo{}, err
	}

	// List pods with the same label as the service
	podList, err := listPodsBySelector(ctx, k8sClient, map[string]string{serviceLabelKey: serviceLabelValue}, netInfo.Namespace)
	if err != nil {
		logger.Error(err, "Failed to list pods with the same label as the service", "label", fmt.Sprintf("%s: %s", serviceLabelKey, serviceLabelValue))
		return nil, utils.NetInfo{}, utils.NetInfo{}, err
	}

	for _, pod := range podList {
		if pod.Name == netInfo.Name {
			continue
		}

		serviceName, err := getServiceName(ctx, k8sClient, pod.Namespace, &pod)
		if err != nil {
			return nil, utils.NetInfo{}, utils.NetInfo{}, err
		}

		otherInfo = utils.NetInfo{
			Name:           pod.Name,
			Namespace:      pod.Namespace,
			Labels:         pod.Labels,
			ContainerPorts: getContainerPorts(&pod),
			Protocol:       getProtocols(&pod),
			ServiceName:    serviceName,
			RelatedEnv:     getRelatedEnv(&pod),
		}
		otherLabelKey, otherLabelValue := getFirstLabelmapversion(otherInfo.Labels)
		otherLabels := fmt.Sprintf("%s: %s", otherLabelKey, otherLabelValue)

		for _, entity := range entities {
			if entity.Type == "LABEL" && entity.Text != otherLabels {
				intent = strings.ReplaceAll(intent, entity.Text+" pod", fmt.Sprintf("pod labeled '%s'", otherLabels))
			}
		}

		if otherInfo.Name != "" {
			if trafficDirection == "ingress" {
				finalIntent := strings.Replace(intent, "traffic", fmt.Sprintf("traffic from '%s pod' labeled '%s'", otherInfo.Name, otherLabels), 1)
				if !strings.Contains(finalIntent, "traffic over") {
					finalIntent = strings.Replace(finalIntent, "traffic", fmt.Sprintf("traffic over %s/%s", netInfo.ContainerPorts, netInfo.Protocol), -1)
				}
				finalIntents = append(finalIntents, finalIntent)
			} else if trafficDirection == "egress" {
				finalIntent := strings.Replace(intent, "traffic", fmt.Sprintf("traffic to '%s pod' labeled '%s'", otherInfo.Name, otherLabels), 1)
				if !strings.Contains(finalIntent, "traffic over") {
					finalIntent = strings.Replace(finalIntent, "traffic", fmt.Sprintf("traffic over %s/%s", netInfo.ContainerPorts, netInfo.Protocol), -1)
				}

				finalIntents = append(finalIntents, finalIntent)
			}
		}
	}

	if len(finalIntents) == 0 {
		logger.Info("No related pods found for the provided service label")
	} else {
		logger.Info("Prompts created", "Network Prompts", finalIntents)
	}

	return finalIntents, netInfo, otherInfo, nil
}

// getServiceName extracts the service name from the pod spec or finds a service with matching labels.
func getServiceName(ctx context.Context, k8sClient client.Client, namespace string, pod *v1.Pod) (string, error) {
	// First try to find service name from environment variables
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.HasSuffix(env.Name, "_ADDR") {
				return strings.Split(env.Value, ":")[0], nil
			}
		}
	}

	// If not found, try to find a service with matching labels
	serviceList := &v1.ServiceList{}
	if err := k8sClient.List(ctx, serviceList, &client.ListOptions{Namespace: namespace}); err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range serviceList.Items {
		if isLabelsMatch(pod.GetLabels(), service.Spec.Selector) {
			return service.Name, nil
		}
	}

	return "", fmt.Errorf("service name not found for pod %s", pod.Name)
}

// Check if labels match
func isLabelsMatch(podLabels, serviceSelector map[string]string) bool {
	for key, value := range serviceSelector {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}

// getRelatedEnv extracts the environment variables from the pod spec and returns them as a slice of Service.
func getRelatedEnv(pod *v1.Pod) []utils.Service {
	var relatedEnv []utils.Service
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.Contains(env.Value, ":") {
				parts := strings.Split(env.Value, ":")
				relatedEnv = append(relatedEnv, utils.Service{Name: parts[0], Port: parts[1]})
			}
		}
	}
	return relatedEnv
}

// getContainerPorts extracts the container ports from the pod spec and returns them as a comma-separated string.
func getContainerPorts(pod *v1.Pod) string {
	var ports []string
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			ports = append(ports, fmt.Sprintf("%d", port.ContainerPort))
		}
	}
	return strings.Join(ports, ", ")
}

// getProtocols extracts the protocols used in the pod containers and returns them as a comma-separated string.
func getProtocols(pod *v1.Pod) string {
	var protocols []string
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			protocols = append(protocols, string(port.Protocol))
		}
	}
	return strings.Join(protocols, ", ")
}

// getEnvVars extracts the environment variables from the pod spec and returns them as a map.
func getEnvVars(pod *v1.Pod) map[string]string {
	envVars := make(map[string]string)
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			envVars[env.Name] = env.Value
		}
	}
	return envVars
}

func executeCommand(ctx context.Context, cmd string) (string, error) {
	output, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v, output: %s", err, string(output))
	}
	return string(output), nil
}

func getTrafficDirection(entities []utils.Entity) string {
	for _, entity := range entities {
		if entity.Type == "TRAFFIC_DIRECTION" {
			direction := strings.ToLower(entity.Text)
			if direction == "ingress" || direction == "income" || direction == "incoming" || direction == "inbound" {
				return "ingress"
			}
			return "egress"
		}
	}
	return ""
}

func getServiceLabel(ctx context.Context, k8sClient client.Client, namespace string, serviceName string) (string, string, error) {
	service := &v1.Service{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: serviceName}, service); err != nil {
		return "", "", fmt.Errorf("failed to fetch service: %w", err)
	}

	// Assuming the service has a selector to identify its pods
	for key, value := range service.Spec.Selector {
		return key, value, nil
	}

	return "", "", fmt.Errorf("no selector found for service: %s", serviceName)
}
