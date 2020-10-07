package k8s

import (
	"context"
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Check ...
func Check() (err error) {
	ctx := context.Background()
	kubeconfig := flag.String("kubeconfig", "/Users/svetomir.smiljkovic/.kube/config", "kubeconfig file")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("The kubeconfig cannot be loaded: %v\n", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Client cannot be set: %v\n", err)
		return err
	}

	pod, err := clientset.CoreV1().Pods("svetomir").Get(ctx, "bee-0", metav1.GetOptions{})
	if err != nil {
		return err
	}
	fmt.Println(pod.Name)

	return
}
