package watcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetConfig(ctx context.Context, k8sClient client.Client, namespace string, name string) (*utils.NetInfo, error) {
	pod := &v1.Pod{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, pod); err != nil {
		return nil, fmt.Errorf("failed to fetch pod: %w", err)
	}

	serviceName, err := getServiceName(ctx, k8sClient, namespace, pod)
	if err != nil {
		return nil, err
	}

	netInfo := &utils.NetInfo{
		Name:           name,
		Namespace:      namespace,
		Labels:         filterLabels(pod.GetLabels()),
		ContainerPorts: getContainerPorts(pod),
		Protocol:       getProtocols(pod),
		ServiceName:    serviceName,
		RelatedEnv:     getRelatedEnv(pod),
	}

	return netInfo, nil
}

func getServiceName(ctx context.Context, k8sClient client.Client, namespace string, pod *v1.Pod) (string, error) {
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.HasSuffix(env.Name, "_ADDR") {
				return strings.Split(env.Value, ":")[0], nil
			}
		}
	}

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

func isLabelsMatch(podLabels, serviceSelector map[string]string) bool {
	for key, value := range serviceSelector {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}

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

func filterLabels(labels map[string]string) map[string]string {
	filteredLabels := make(map[string]string)
	for key, value := range labels {
		if key != "pod-template-hash" {
			filteredLabels[key] = value
		}
	}
	return filteredLabels
}

func getContainerPorts(pod *v1.Pod) string {
	var ports []string
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			ports = append(ports, fmt.Sprintf("%d", port.ContainerPort))
		}
	}
	return strings.Join(ports, ", ")
}

func getProtocols(pod *v1.Pod) string {
	var protocols []string
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			protocols = append(protocols, string(port.Protocol))
		}
	}
	return strings.Join(protocols, ", ")
}
