/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	emailv1 "github.com/erickcezar/mailerlite-operator/api/v1"
)

const (
	mailgunURL    = "https://api.mailgun.net/v4/domains"
	mailersendURL = "https://api.mailersend.com/v1/domains"
)

// EmailSenderConfigReconciler reconciles a EmailSenderConfig object
type EmailSenderConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emailsenderconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emailsenderconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emailsenderconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=*
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *EmailSenderConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithName("emailsenderconfig")

	// Fetch the EmailSenderConfig instance
	config := &emailv1.EmailSenderConfig{}
	err := r.Get(ctx, req.NamespacedName, config)
	if err != nil {
		log.Error(err, "unable to fetch EmailSenderConfig")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Confirm the email sending settings
	// For simplicity, just log the creation or update
	log.Info("EmailSenderConfig created or updated", "name", config.Name)

	apiToken, err := getSecret(ctx, config.Spec.APITokenSecretRef, r.Client, log, req.Namespace)

	if err != nil {
		config.Status.Status = "Failed"
		config.Status.Error = "Secret not found"
	} else {
		if strings.Contains(config.Spec.SenderEmail, "mlsender.net") {
			config.Status.Status, config.Status.Error = checkTokenMail("mailersend", apiToken, mailersendURL)
		} else if strings.Contains(config.Spec.SenderEmail, "mailgun.org") {
			config.Status.Status, config.Status.Error = checkTokenMail("mailgun", apiToken, mailgunURL)
		}
	}

	err = r.Status().Update(ctx, config)
	if err != nil {
		log.Error(err, "unable to update EmailSenderConfig status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmailSenderConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&emailv1.EmailSenderConfig{}).
		Complete(r)
}

func checkTokenMail(provider, apiToken, url string) (string, string) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "Failed", "Failed request provider api"
	}

	if provider == "mailgun" {
		req.SetBasicAuth("api", apiToken)
	} else if provider == "mailersend" {
		req.Header.Add("Authorization", "Bearer "+apiToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "Failed", "Failed request provider api"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "Ready", ""
	}

	return "Failed", "Failed auth using apiToken"
}
