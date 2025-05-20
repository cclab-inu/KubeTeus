// property.go
package validator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PropertyValidator(ctx context.Context, k8sClient client.Client, policies interface{}) []error {
	var errs []error
	// logger := log.FromContext(ctx)

	for _, policy := range policies.([]interface{}) {
		policyMap, err := toMap(policy)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid policy type: %v", err))
			continue
		}

		kind, ok := policyMap["kind"].(string)
		if !ok {
			errs = append(errs, fmt.Errorf("missing or invalid kind in policy"))
			continue
		}

		metadata, ok := policyMap["metadata"].(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("missing or invalid metadata in policy"))
			continue
		}

		spec, ok := policyMap["spec"].(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("missing or invalid spec in policy"))
			continue
		}
		// logger.Info("spec", "spec", spec)

		selector, ok := spec["selector"].(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("missing or invalid selector in policy"))
			continue
		}

		matchLabels, ok := selector["matchLabels"].(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("missing or invalid matchLabels in policy"))
			continue
		}

		namespace, ok := metadata["namespace"].(string)
		if !ok {
			errs = append(errs, fmt.Errorf("missing or invalid namespace in policy"))
			continue
		}

		labelMap := make(map[string]string)
		for key, value := range matchLabels {
			labelMap[key] = value.(string)
		}

		switch kind {
		case "CiliumNetworkPolicy":
			err := validateNetworkPolicy(ctx, k8sClient, namespace, labelMap, spec)
			if err != nil {
				errs = append(errs, err)
			}
		case "KubeArmorPolicy":
			err := validateSystemPolicy(ctx, k8sClient, namespace, labelMap, spec)
			if err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, fmt.Errorf("unknown policy kind: %s", kind))
		}
	}

	return errs
}

func validateNetworkPolicy(ctx context.Context, k8sClient client.Client, namespace string, matchLabels map[string]string, spec map[string]interface{}) error {
	rules, ok := spec["rules"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid rules format in network policy")
	}

	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid rule format in network policy")
		}
		if from, ok := ruleMap["from"]; ok {
			if err := validatePortListening(ctx, k8sClient, namespace, matchLabels, from.([]interface{})); err != nil {
				return err
			}
		}
		if to, ok := ruleMap["to"]; ok {
			if err := validatePortListening(ctx, k8sClient, namespace, matchLabels, to.([]interface{})); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateSystemPolicy(ctx context.Context, k8sClient client.Client, namespace string, matchLabels map[string]string, spec map[string]interface{}) error {
	var validationErrors []error

	// Process validation
	if process, ok := spec["process"].(map[string]interface{}); ok && len(process) != 0 {
		if err := validateProcessExistence(ctx, k8sClient, namespace, matchLabels, process); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	// File validation
	if file, ok := spec["file"].(map[string]interface{}); ok && len(file) != 0 {
		if err := validateFileExistence(file); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	// Syscalls validation
	if syscall, ok := spec["syscalls"].(map[string]interface{}); ok && len(syscall) != 0 {
		if err := validateSystemCalls(syscall); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	// Combine errors from all validations
	if len(validationErrors) > 0 {
		return fmt.Errorf("validation errors: %v", validationErrors)
	}
	return nil
}

func validatePortListening(ctx context.Context, k8sClient client.Client, namespace string, matchLabels map[string]string, rules []interface{}) error {
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid rule format in port listening")
		}
		if port, ok := ruleMap["port"]; ok {
			if protocol, ok := ruleMap["protocol"]; ok {
				expectedPort, err := strconv.Atoi(port.(string))
				if err != nil {
					return errors.Errorf("error parsing expected port: %v", err)
				}

				var pods corev1.PodList
				listOpts := []client.ListOption{
					client.InNamespace(namespace),
					client.MatchingLabels(matchLabels),
				}
				if err := k8sClient.List(ctx, &pods, listOpts...); err != nil {
					return errors.Errorf("error fetching pods: %v", err)
				}
				if len(pods.Items) == 0 {
					return errors.Errorf("no pods found matching the selector in namespace %s", namespace)
				}

				portFound := false
				for _, pod := range pods.Items {
					for _, container := range pod.Spec.Containers {
						for _, p := range container.Ports {
							if p.ContainerPort == int32(expectedPort) && strings.EqualFold(string(p.Protocol), protocol.(string)) {
								portFound = true
								break
							}
						}
						if portFound {
							break
						}
					}
					if portFound {
						break
					}
				}
				if !portFound {
					return errors.Errorf("no containers found in namespace %s with labels %v listening on the expected port %d with protocol %s", namespace, matchLabels, expectedPort, protocol)
				}
			} else {
				return errors.New("protocol is missing in rule")
			}
		} else {
			return errors.New("port is missing in rule")
		}
	}
	return nil
}

func validateProcessExistence(ctx context.Context, k8sClient client.Client, namespace string, matchLabels map[string]string, process map[string]interface{}) error {
	expectedProcesses, ok := process["matchPaths"]
	if !ok {
		return fmt.Errorf("missing process matchPaths")
	}

	// Ensure expectedProcesses is a slice of strings
	expectedProcessesSlice, ok := expectedProcesses.([]interface{})
	if !ok {
		return fmt.Errorf("invalid process matchPaths format")
	}
	expectedProcessesStr := make([]string, len(expectedProcessesSlice))
	for i, proc := range expectedProcessesSlice {
		expectedProcessesStr[i], ok = proc.(string)
		if !ok {
			return fmt.Errorf("process matchPaths should be a list of strings")
		}
	}

	var pods corev1.PodList
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(matchLabels),
	}
	if err := k8sClient.List(ctx, &pods, listOpts...); err != nil {
		return errors.Errorf("error fetching pods: %v", err)
	}
	if len(pods.Items) == 0 {
		return errors.Errorf("no pods found matching the selector in namespace %s", namespace)
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			cmd := exec.Command("kubectl", "exec", pod.Name, "-n", namespace, "-c", container.Name, "--", "ps", "-ef")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return errors.Errorf("error executing command to check processes in pod %s, container %s: %v", pod.Name, container.Name, err)
			}
			for _, proc := range expectedProcessesStr {
				if !strings.Contains(string(output), proc) {
					return errors.Errorf("process %s not found in pod %s, container %s", proc, pod.Name, container.Name)
				}
			}
		}
	}

	return nil
}

func validateFileExistence(file map[string]interface{}) error {
	matchPaths, ok := file["matchPaths"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid file matchPaths format")
	}

	for _, pathInterface := range matchPaths {
		pathMap, ok := pathInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected a map for file path but got type %T", pathInterface)
		}
		path, ok := pathMap["path"].(string)
		if !ok {
			return fmt.Errorf("expected a string for file path but got type %T in map", pathMap["path"])
		}
		fmt.Printf("Checking if file or directory exists: %s\n", path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.Errorf("file or directory does not exist: %s", path)
		}
	}
	return nil
}

func validateSystemCalls(syscall map[string]interface{}) error {
	matchPaths, ok := syscall["matchPaths"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid syscall matchPaths format")
	}

	knownSystemCalls := []string{"open", "read", "write", "close", "unlink", "rmdir"}
	for _, path := range matchPaths {
		syscalls := strings.Split(path.(string), ",")
		for _, call := range syscalls {
			if !contains(knownSystemCalls, strings.TrimSpace(call)) {
				return errors.Errorf("invalid system call: %s", call)
			}
		}
	}
	return nil
}

// Helper function to check if a slice contains a specific string.
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
