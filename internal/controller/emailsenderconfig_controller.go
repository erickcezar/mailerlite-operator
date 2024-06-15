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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	emailv1 "github.com/erickcezar/mailerlite-operator/api/v1"
	"github.com/go-logr/logr"
)

// EmailSenderConfigReconciler reconciles a EmailSenderConfig object
type EmailSenderConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emailsenderconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emailsenderconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emailsenderconfigs/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *EmailSenderConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("emailsenderconfig", req.NamespacedName)

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

	_, err = getSecret(ctx, config.Spec.APITokenSecretRef, r.Client, log, req.Namespace)

	if err != nil {
		config.Status.Status = "Secret not found"
	} else {
		config.Status.Status = "Ready"
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
