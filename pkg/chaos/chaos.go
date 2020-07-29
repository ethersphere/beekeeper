package chaos

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/dynamick8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func CheckChaosMesh(ctx context.Context, kubeconfig string, namespace string) (err error) {
	kubeRes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	client, err := dynamick8s.NewClient(kubeconfig, "chaos-testing", kubeRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	_, err = client.Get(ctx, "chaos-mesh-controller-manager")
	if err != nil {
		return fmt.Errorf("error getting chaos-mesh-controller-manager service: %+v", err)
	}
	return
}

func PodFailure(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "podchaos"}

	var label string

	if podname == "bee" {
		label = "app.kubernetes.io/name"
	} else {
		label = "statefulset.kubernetes.io/pod-name"
	}

	object := podFailure(mode, value, namespace, label, podname, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "pod-failure-"+mode+"-"+podname)
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

	if podname == "bee" {
		label = "app.kubernetes.io/name"
	} else {
		label = "statefulset.kubernetes.io/pod-name"
	}

	if cron != "" {
		object = podKillCron(mode, value, namespace, label, podname, cron)
	} else {
		object = podKill(mode, value, namespace, label, podname)
	}

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "pod-kill-"+mode+"-"+podname)
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

	if podname1 == "bee" {
		label1 = "app.kubernetes.io/name"
	} else {
		label1 = "statefulset.kubernetes.io/pod-name"
	}

	if podname2 == "bee" {
		label2 = "app.kubernetes.io/name"
	} else {
		label2 = "statefulset.kubernetes.io/pod-name"
	}

	if direction == "" {
		direction = "both"
	}

	object := networkPartition(mode1, value1, mode2, value2, namespace, label1, podname1, label2, podname2, direction, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "network-partition-"+mode1+"-"+podname1)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkLoss(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, loss string, correlation string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname == "bee" {
		label = "app.kubernetes.io/name"
	} else {
		label = "statefulset.kubernetes.io/pod-name"
	}

	object := networkLoss(mode, value, namespace, label, podname, loss, correlation, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "network-loss-"+mode+"-"+podname)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkDelay(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, latency string, correlation string, jitter string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname == "bee" {
		label = "app.kubernetes.io/name"
	} else {
		label = "statefulset.kubernetes.io/pod-name"
	}

	object := networkDelay(mode, value, namespace, label, podname, latency, correlation, jitter, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "network-delay-"+mode+"-"+podname)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkDuplicate(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, duplicate string, correlation string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname == "bee" {
		label = "app.kubernetes.io/name"
	} else {
		label = "statefulset.kubernetes.io/pod-name"
	}

	object := networkDuplicate(mode, value, namespace, label, podname, duplicate, correlation, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "network-duplicate-"+mode+"-"+podname)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

func NetworkCorrupt(ctx context.Context, kubeconfig string, action string, mode string, value string, namespace string, podname string, corrupt string, correlation string, duration string, cron string) (err error) {
	chaosRes := schema.GroupVersionResource{Group: "pingcap.com", Version: "v1alpha1", Resource: "networkchaos"}

	var label string

	if podname == "bee" {
		label = "app.kubernetes.io/name"
	} else {
		label = "statefulset.kubernetes.io/pod-name"
	}

	object := networkCorrupt(mode, value, namespace, label, podname, corrupt, correlation, duration, cron)

	client, err := dynamick8s.NewClient(kubeconfig, namespace, chaosRes)
	if err != nil {
		fmt.Printf("error: %+v", err)
	}
	if action == "create" {
		err = client.Create(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "update" {
		err = client.Update(ctx, object)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	if action == "delete" {
		err = client.Delete(ctx, "network-corrupt-"+mode+"-"+podname)
		if err != nil {
			fmt.Printf("error: %+v", err)
		}
	}
	return
}

// func BeeReplicaSet(ctx context.Context, kubeconfig string, namespace string, replica int64) (err error) {
// 	kubeRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
// 	client, err := dynamick8s.NewClient(kubeconfig, namespace, kubeRes)
// 	if err != nil {
// 		fmt.Printf("error: %+v", err)
// 	}
// 	_ = client.UpdateBeeReplica(ctx, replica)
// 	if err != nil {
// 		return fmt.Errorf("error getting chaos-mesh-controller-manager service: %+v", err)
// 	}
// 	return
// }
