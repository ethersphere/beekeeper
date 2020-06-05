package chaos

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func networkPartition(mode1 string, value1 string, mode2 string, value2 string, namespace string, label1 string, podname1 string, label2 string, podname2 string, direction string, duration string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "NetworkChaos",
			"metadata": map[string]interface{}{
				"name":      "network-partition-" + mode1 + "-" + podname1,
				"namespace": "chaos-testing",
			},
			"spec": map[string]interface{}{
				"action":   "partition",
				"mode":     mode1,
				"value":    value1,
				"duration": duration,
				"selector": map[string]interface{}{
					"namespaces": []string{
						namespace},
					"labelSelectors": map[string]interface{}{
						label1: podname1,
					},
				},
				"direction": direction,
				"target": map[string]interface{}{
					"selector": map[string]interface{}{
						"namespaces": []string{
							namespace},
						"labelSelectors": map[string]interface{}{
							label2: podname2,
						},
					},
					"mode":  mode2,
					"value": value2,
					"scheduler": map[string]interface{}{
						"cron": "@every " + cron,
					},
				},
			},
		},
	}
}

func networkLoss(mode string, value string, namespace string, label string, podname string, loss string, correlation string, duration string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "NetworkChaos",
			"metadata": map[string]interface{}{
				"name":      "network-loss-" + mode + "-" + podname,
				"namespace": "chaos-testing",
			},
			"spec": map[string]interface{}{
				"action":   "loss",
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
				"loss": map[string]interface{}{
					"loss":        loss,
					"correlation": correlation,
				},
				"scheduler": map[string]interface{}{
					"cron": "@every " + cron,
				},
			},
		},
	}
}

func networkDelay(mode string, value string, namespace string, label string, podname string, latency string, correlation string, jitter string, duration string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "NetworkChaos",
			"metadata": map[string]interface{}{
				"name":      "network-delay-" + mode + "-" + podname,
				"namespace": "chaos-testing",
			},
			"spec": map[string]interface{}{
				"action":   "delay",
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
				"delay": map[string]interface{}{
					"latency":     latency,
					"correlation": correlation,
					"jitter":      jitter,
				},
				"scheduler": map[string]interface{}{
					"cron": "@every " + cron,
				},
			},
		},
	}
}

func networkDuplicate(mode string, value string, namespace string, label string, podname string, duplicate string, correlation string, duration string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "NetworkChaos",
			"metadata": map[string]interface{}{
				"name":      "network-duplicate-" + mode + "-" + podname,
				"namespace": "chaos-testing",
			},
			"spec": map[string]interface{}{
				"action":   "duplicate",
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
				"duplicate": map[string]interface{}{
					"duplicate":   duplicate,
					"correlation": correlation,
				},
				"scheduler": map[string]interface{}{
					"cron": "@every " + cron,
				},
			},
		},
	}
}

func networkCorrupt(mode string, value string, namespace string, label string, podname string, corrupt string, correlation string, duration string, cron string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pingcap.com/v1alpha1",
			"kind":       "NetworkChaos",
			"metadata": map[string]interface{}{
				"name":      "network-corrupt-" + mode + "-" + podname,
				"namespace": "chaos-testing",
			},
			"spec": map[string]interface{}{
				"action":   "corrupt",
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
				"corrupt": map[string]interface{}{
					"corrupt":     corrupt,
					"correlation": correlation,
				},
				"scheduler": map[string]interface{}{
					"cron": "@every " + cron,
				},
			},
		},
	}
}
