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
	clientset kubernetes.Interface
}

// NewClient constructs a new Client.
func NewClient(clientset kubernetes.Interface) *Client {
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

// Set updates ServiceAccount or creates it if it does not exist
func (c *Client) Set(ctx context.Context, name, namespace string, o Options) (sa *v1.ServiceAccount, err error) {
	spec := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		AutomountServiceAccountToken: &o.AutomountServiceAccountToken,
		ImagePullSecrets: func() (l []v1.LocalObjectReference) {
			for _, s := range o.ImagePullSecrets {
				l = append(l, v1.LocalObjectReference{Name: s})
			}
			return l
		}(),
		Secrets: func() (l []v1.ObjectReference) {
			for _, s := range o.Secrets {
				l = append(l, v1.ObjectReference{Name: s})
			}
			return l
		}(),
	}

	sa, err = c.clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, spec, metav1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			sa, err = c.clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, spec, metav1.CreateOptions{})
			if err != nil {
				return nil, fmt.Errorf("creating service account %s in namespace %s: %w", name, namespace, err)
			}
		} else {
			return nil, fmt.Errorf("updating service account %s in namespace %s: %w", name, namespace, err)
		}
	}

	return sa, err
}

// Delete deletes ServiceAccount
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	err = c.clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting service account %s in namespace %s: %w", name, namespace, err)
	}

	return err
}
