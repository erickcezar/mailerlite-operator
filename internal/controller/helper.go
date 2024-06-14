package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getSecret(ctx context.Context, secretName string, k8sClient client.Client, log logr.Logger, namespace string) (string, error) {

	log = log.WithValues("secret", secretName, "namespace", namespace)

	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	if err != nil {
		log.Error(err, "unable to fetch secret")
		//		email.Status.DeliveryStatus = "Failed"
		//		email.Status.Error = "Secret not found"
		//		r.Status().Update(ctx, email)
		return "", client.IgnoreNotFound(err)
	}

	apiToken, exists := secret.Data["apiToken"]
	if !exists {
		err := fmt.Errorf("apiToken not found in secret %s/%s", namespace, secretName)
		log.Error(err, "apiToken key missing in secret data")
		return "", err
	}

	return string(apiToken), nil
}
