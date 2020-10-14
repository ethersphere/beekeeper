package bee

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
)

// Options ...
type Options struct {
	Namespace string
}

// Check ...
func Check(clientset *kubernetes.Clientset, o Options) (err error) {
	ctx := context.Background()
	fmt.Println(ctx)

	// configuration
	// cmOptions.Namespace = o.Namespace
	// if err := configmap.Set(ctx, clientset, cmOptions); err != nil {
	// 	return fmt.Errorf("set configmap: %s", err)
	// }

	// secOptions.Namespace = o.Namespace
	// if err := secret.Set(ctx, clientset, secOptions); err != nil {
	// 	return fmt.Errorf("set secret: %s", err)
	// }

	// // services
	// saOptions.Namespace = o.Namespace
	// if err := serviceaccount.Set(ctx, clientset, saOptions); err != nil {
	// 	return fmt.Errorf("set serviceaccount %s", err)
	// }

	// svcOptions.Namespace = o.Namespace
	// if err := service.Set(ctx, clientset, svcOptions); err != nil {
	// 	return fmt.Errorf("set service %s", err)
	// }

	// headlessSvcOptions.Namespace = o.Namespace
	// if err := services.Set(ctx, clientset, headlessSvcOptions); err != nil {
	// 	return fmt.Errorf("set service %s", err)
	// }

	// // ingress
	// ingressOptions.Namespace = o.Namespace
	// if err := ingress.Set(ctx, clientset, ingressOptions); err != nil {
	// 	return fmt.Errorf("set ingress %s", err)
	// }

	// // statefulset
	// ssOptions.Namespace = o.Namespace
	// if err := statefulset.Set(ctx, clientset, ssOptions); err != nil {
	// 	return fmt.Errorf("set statefulset %s", err)
	// }

	return
}
