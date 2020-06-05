package chaos

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/dynamick8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func PodFailure(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "podchaos"}

	var label string
	var object *unstructured.Unstructured

	if podname != "" {
		label = "statefulset.kubernetes.io/pod-name"
	} else {
		label = "app.kubernetes.io/name"
		podname = "bee"
	}

	object = podFailure(mode, value, namespace, label, podname, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "pod-failure-"+mode)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func PodKill(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "podchaos"}

	var label string
	var object *unstructured.Unstructured

	if podname != "" {
		label = "statefulset.kubernetes.io/pod-name"
	} else {
		label = "app.kubernetes.io/name"
		podname = "bee"
	}

	if cron != "" {
		object = podKillCron(mode, value, namespace, label, podname, cron)
	} else {
		object = podKill(mode, value, namespace, label, podname)
	}

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "pod-kill-"+mode)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkPartition(ctx context.Context, kubeconfig string, action string, mode1 string, value1 string, mode2 string, value2 string, namespace string, podname1 string, podname2 string, direction string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label1 string
	var label2 string

	if podname1 != "" {
		label1 = "statefulset.kubernetes.io/pod-name"
	} else {
		label1 = "app.kubernetes.io/name"
		podname1 = "bee"
	}

	if podname2 != "" {
		label2 = "statefulset.kubernetes.io/pod-name"
	} else {
		label2 = "app.kubernetes.io/name"
		podname2 = "bee"
	}

	if direction == "" {
		direction = "both"
	}

	object := networkPartition(mode1, value1, mode2, value2, namespace, label1, podname1, label2, podname2, direction, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "network-partition-"+mode1)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkLoss(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, loss string, correlation string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname != "" {
		label = "statefulset.kubernetes.io/pod-name"
	} else {
		label = "app.kubernetes.io/name"
		podname = "bee"
	}

	object := networkLoss(mode, value, namespace, label, podname, loss, correlation, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "network-loss-"+mode)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkDelay(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, latency string, correlation string, jitter string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname != "" {
		label = "statefulset.kubernetes.io/pod-name"
	} else {
		label = "app.kubernetes.io/name"
		podname = "bee"
	}

	object := networkDelay(mode, value, namespace, label, podname, latency, correlation, jitter, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "network-delay-"+mode)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkDuplicate(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, duplicate string, correlation string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname != "" {
		label = "statefulset.kubernetes.io/pod-name"
	} else {
		label = "app.kubernetes.io/name"
		podname = "bee"
	}

	object := networkDuplicate(mode, value, namespace, label, podname, duplicate, correlation, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "network-duplicate-"+mode)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkCorrupt(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, corrupt string, correlation string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname != "" {
		label = "statefulset.kubernetes.io/pod-name"
	} else {
		label = "app.kubernetes.io/name"
		podname = "bee"
	}

	object := networkCorrupt(mode, value, namespace, label, podname, corrupt, correlation, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	// TODO: needs resourceVersion
	// if action == "update" {
	// 	err = client.Update(ctx, object)
	// if err != nil {
	// 	fmt.Printf("error: %+v", err)
	// }
	// }
	if action == "delete" {
		err = client.Delete(ctx, "network-corrupt-"+mode)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}
