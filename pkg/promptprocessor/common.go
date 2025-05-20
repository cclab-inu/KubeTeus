package promptprocessor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"github.com/cclab-inu/KubeTeus/pkg/watcher"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func isPodMatch(podName, name string) bool {
	// Allow prefix-based matching
	return strings.HasPrefix(podName, name)
}

/*
func isPodMatch(podName, name string) bool {
	if strings.Contains(podName, name) {
		podParts := strings.Split(podName, "-")
		nameParts := strings.Split(name, "-")

		for i := range nameParts {
			if i >= len(podParts) || podParts[i] != nameParts[i] {
				return false
			}
		}
		return true
	}
	return false
}*/

func listPodsBySelector(ctx context.Context, k8sClient client.Client, selector map[string]string, namespace string) ([]v1.Pod, error) {
	var podList v1.PodList
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector),
		Namespace:     namespace,
	}
	if err := k8sClient.List(ctx, &podList, listOpts); err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func listPodsByNamespace(ctx context.Context, k8sClient client.Client, namespace string) ([]v1.Pod, error) {
	var podList v1.PodList
	listOpts := &client.ListOptions{
		Namespace: namespace,
	}
	if err := k8sClient.List(ctx, &podList, listOpts); err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func filterPodsByName(pods []v1.Pod, name string) []v1.Pod {
	var filtered []v1.Pod
	for _, pod := range pods {
		if isPodMatch(pod.Name, name) {
			filtered = append(filtered, pod)
		}
	}
	return filtered
}

func formatLabels(labels map[string]string) string {
	var labelStrings []string
	for key, value := range labels {
		labelStrings = append(labelStrings, fmt.Sprintf("'%s: %s'", key, value))
	}
	return strings.Join(labelStrings, ", ")
}

func joinStrings(items []string, delimiter string) string {
	if len(items) == 0 {
		return ""
	}

	result := items[0]
	for _, item := range items[1:] {
		result += delimiter + " " + item
	}
	return result
}

func extractContainerPorts(containers []v1.Container) string {
	var ports []string
	for _, container := range containers {
		for _, port := range container.Ports {
			ports = append(ports, fmt.Sprintf("%d", port.ContainerPort))
		}
	}
	return strings.Join(ports, ", ")
}

func extractProtocols(containers []v1.Container) string {
	var protocols []string
	for _, container := range containers {
		for _, port := range container.Ports {
			protocols = append(protocols, string(port.Protocol))
		}
	}
	return strings.Join(protocols, ", ")
}

func getFirstLabel(pod *v1.Pod) (string, string) {
	for key, value := range pod.Labels {
		if key != "pod-template-hash" {
			return key, value
		}
	}
	return "", ""
}

func getFirstLabelmapversion(labels map[string]string) (string, string) {
	for key, value := range labels {
		if key != "pod-template-hash" {
			return key, value
		}
	}
	return "", ""
}

func connectToDB() (*sql.DB, error) {
	return sql.Open("mysql", "cclab:cclab@tcp(localhost:3306)/traffic_db")
}

func containsAnotherPod(entities []utils.Entity) bool {
	podCount := 0
	for _, entity := range entities {
		if entity.Type == "POD_NAME" && entity.Text != "" {
			podCount++
		}
	}
	return podCount > 1
}

func findBasePod(podName, namespace string, db *sql.DB) (string, string, string, error) {
	var srcPod, dstPod, direction string
	query := `SELECT src_pod, dst_pod, direction FROM pod_traffic WHERE src_pod LIKE ? OR dst_pod LIKE ? LIMIT 1`
	err := db.QueryRow(query, "%/"+podName, "%/"+podName).Scan(&srcPod, &dstPod, &direction)
	if err != nil {
		return "", "", "", err
	}
	return srcPod, dstPod, direction, nil
}

func extractPodName(fullPodName string) string {
	parts := strings.Split(fullPodName, "/")
	if len(parts) == 2 {
		return parts[1]
	}
	return fullPodName
}

func getPodNetInfo(ctx context.Context, k8sClient client.Client, shortPodName, namespace string) (utils.NetInfo, error) {
	pods, err := listPodsByNamespace(ctx, k8sClient, namespace)
	if err != nil {
		return utils.NetInfo{}, err
	}

	matchingPods := filterPodsByName(pods, shortPodName)
	if len(matchingPods) == 0 {
		return utils.NetInfo{}, fmt.Errorf("no matching pod found for %s", shortPodName)
	}
	if len(matchingPods) > 1 {
		return utils.NetInfo{}, fmt.Errorf("multiple pods matched for %s", shortPodName)
	}

	fullPodName := matchingPods[0].Name
	netInfoPtr, err := watcher.GetConfig(ctx, k8sClient, namespace, fullPodName)
	if err != nil {
		return utils.NetInfo{}, err
	}
	return *netInfoPtr, nil
}

/*func getPodNetInfo(ctx context.Context, k8sClient client.Client, podName, namespace string) (utils.NetInfo, error) {
	netInfoPtr, err := watcher.GetConfig(ctx, k8sClient, namespace, podName)
	if err != nil {
		return utils.NetInfo{}, err
	}
	return *netInfoPtr, nil
}*/

func extractLabel(labels map[string]string) (string, string) {
	for key, value := range labels {
		return key, value
	}
	return "", ""
}

func findPodByIP(ctx context.Context, ip string, namespace string) string {
	findCmd := fmt.Sprintf("kubectl get pod -n %s -o wide | grep %s | awk '{print $1}'", namespace, ip)
	output, err := executeCommand(ctx, findCmd)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to find pod by IP", "IP", ip)
		return ""
	}
	return strings.TrimSpace(output)
}
