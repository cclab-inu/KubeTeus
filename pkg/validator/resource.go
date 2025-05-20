package validator

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ResourceValidator(ctx context.Context, k8sClient client.Client, logger logr.Logger, policies interface{}) []error {
	var errs []error

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

		selectorLabels := labels.SelectorFromSet(labelMap)

		// Validate namespace existence and status
		err = validateNamespaces(ctx, k8sClient, namespace)
		if err != nil {
			errs = append(errs, err)
		}

		err = validateNamespaceStatus(ctx, k8sClient, namespace)
		if err != nil {
			errs = append(errs, err)
		}

		err = checkPodExistence(ctx, k8sClient, namespace, selectorLabels)
		if err != nil {
			errs = append(errs, err)
		}

		if kind == "CiliumNetworkPolicy" {
			err = validateServices(ctx, k8sClient, namespace, labelMap)
			if err != nil {
				errs = append(errs, err)
			}
		}

	}

	return errs
}

// checkPodExistence checks if the Pods matching the selector exist.
func checkPodExistence(ctx context.Context, k8sClient client.Client, namespace string, selector labels.Selector) error {
	var pods corev1.PodList
	logger := log.FromContext(ctx)

	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}

	if err := k8sClient.List(ctx, &pods, listOpts...); err != nil {
		logger.Error(err, "Failed to list pods", "namespace", namespace, "labels", selector)
		return fmt.Errorf("error listing pods: %v", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no matching pods found in namespace %s with labels %v", namespace, selector)
	}

	for _, pod := range pods.Items {
		logger.Info("Matching pod found", "podName", pod.Name, "namespace", namespace)
	}

	return nil
}

// validateServices checks if the Services matching the selector exist.
func validateServices(ctx context.Context, k8sClient client.Client, namespace string, matchLabels map[string]string) error {
	var services corev1.ServiceList
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(matchLabels),
	}
	if err := k8sClient.List(ctx, &services, listOpts...); err != nil {
		return errors.Errorf("error fetching services: %v", err)
	}
	if len(services.Items) == 0 {
		return errors.Errorf("no services found matching the selector")
	}
	return nil
}

// validateNamespaces checks the existence of the namespace.
func validateNamespaces(ctx context.Context, k8sClient client.Client, namespace string) error {
	var ns corev1.Namespace
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, &ns); err != nil {
		return errors.Errorf("error fetching namespace '%s': %v", namespace, err)
	}
	return nil
}

// validateNamespaceStatus checks the status of the namespace to ensure it is active.
func validateNamespaceStatus(ctx context.Context, k8sClient client.Client, namespace string) error {
	var ns corev1.Namespace
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, &ns); err != nil {
		return errors.Errorf("error fetching namespace '%s': %v", namespace, err)
	}

	if ns.Status.Phase != corev1.NamespaceActive {
		return errors.Errorf("namespace '%s' is not in an active phase", namespace)
	}

	return nil
}
