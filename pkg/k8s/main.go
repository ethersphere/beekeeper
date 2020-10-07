package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Check ...
func Check(clientset *kubernetes.Clientset, namespace string) (err error) {
	ctx := context.Background()

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, "bee-0", metav1.GetOptions{})
	if err != nil {
		return err
	}
	fmt.Println(pod)

	return
}
