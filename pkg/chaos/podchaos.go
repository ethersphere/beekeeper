package chaos

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func podFailure(mode string, value string, namespace string, label string, podname string, duration string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "PodChaos",
			"metadata": map[string]interface{}{
				"name":      "pod-failure-" + mode + "-" + podname,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"action":   "pod-failure",
				"mode":     mode,
				"value":    value,
				"duration": duration,
				"selector": map[string]interface{}{
					"namespaces": []string{
						namespace},
					"labelSelectors": map[string]interface{}{
						label: podname,
					},
				},
				"scheduler": map[string]interface{}{
					"cron": "@every " + cron,
				},
			},
		},
	}
}

func podKillCron(mode string, value string, namespace string, label string, podname string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "PodChaos",
			"metadata": map[string]interface{}{
				"name":      "pod-kill-" + mode + "-" + podname,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"action": "pod-kill",
				"mode":   mode,
				"value":  value,
				"selector": map[string]interface{}{
					"namespaces": []string{
						namespace},
					"labelSelectors": map[string]interface{}{
						label: podname,
					},
				},
				"scheduler": map[string]interface{}{
					"cron": "@every " + cron,
				},
			},
		},
	}
}

func podKill(mode string, value string, namespace string, label string, podname string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "PodChaos",
			"metadata": map[string]interface{}{
				"name":      "pod-kill-" + mode + "-" + podname,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"action": "pod-kill",
				"mode":   mode,
				"value":  value,
				"selector": map[string]interface{}{
					"namespaces": []string{
						namespace},
					"labelSelectors": map[string]interface{}{
						label: podname,
					},
				},
			},
		},
	}
}
