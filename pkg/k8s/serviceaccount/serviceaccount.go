package serviceaccount

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes ServiceAccount.
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient constructs a new Client.
func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations                  map[string]string
	Labels                       map[string]string
	AutomountServiceAccountToken bool
	ImagePullSecrets             []string
	Secrets                      []string
}

// Set creates ServiceAccount, if ServiceAccount already exists updates in place
func (c Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Secrets: func() (l []v1.ObjectReference) {
			if len(o.Secrets) > 0 {
				for _, s := range o.Secrets {
					l = append(l, v1.ObjectReference{Name: s})
				}
			}
			return
		}(),
		ImagePullSecrets: func() (l []v1.LocalObjectReference) {
			if len(o.ImagePullSecrets) > 0 {
				for _, s := range o.ImagePullSecrets {
					l = append(l, v1.LocalObjectReference{Name: s})
				}
			}
			return
		}(),
		AutomountServiceAccountToken: &o.AutomountServiceAccountToken,
	}

	_, err = c.clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			fmt.Printf("service account %s already exists in the namespace %s, updating the service account\n", name, namespace)
			_, err = c.clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}
