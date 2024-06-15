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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	emailv1 "github.com/erickcezar/mailerlite-operator/api/v1"
	"github.com/go-logr/logr"
)

// EmailReconciler reconciles a Email object
type EmailReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emails,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emails/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=email.mailerlite.com,resources=emails/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Email object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *EmailReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("email", req.NamespacedName)

	// Fetch the Email instance
	email := &emailv1.Email{}
	err := r.Get(ctx, req.NamespacedName, email)
	if err != nil {
		log.Error(err, "unable to fetch Email")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the referenced EmailSenderConfig
	config := &emailv1.EmailSenderConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: email.Spec.SenderConfigRef, Namespace: req.Namespace}, config)
	if err != nil {
		log.Error(err, "unable to fetch EmailSenderConfig")
		email.Status.DeliveryStatus = "Failed"
		email.Status.Error = "EmailSenderConfig not found"
		r.Status().Update(ctx, email)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the secret containing the API token
	encodedToken, err := getSecret(ctx, config.Spec.APITokenSecretRef, r.Client, log, req.Namespace)
	if err != nil {
		log.Error(err, "unable to fetch secret")
		email.Status.DeliveryStatus = "Failed"
		email.Status.Error = "Secret not found"
		r.Status().Update(ctx, email)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	apiToken, err := base64.StdEncoding.DecodeString(encodedToken)

	if err != nil {
		log.Error(err, "error decoding secret")
		return ctrl.Result{}, err
	}

	// Send the email via MailerSend
	deliveryStatus, messageID, err := sendEmail(string(apiToken), config.Spec.SenderEmail, email.Spec.RecipientEmail, email.Spec.Subject, email.Spec.Body)
	if err != nil {
		log.Error(err, "failed to send email")
		email.Status.DeliveryStatus = "Failed"
		email.Status.Error = err.Error()
	} else {
		email.Status.DeliveryStatus = deliveryStatus
		email.Status.MessageID = messageID
	}

	err = r.Status().Update(ctx, email)
	if err != nil {
		log.Error(err, "unable to update Email status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func sendEmail(apiToken, senderEmail, recipientEmail, subject, body string) (string, string, error) {
	if strings.Contains(senderEmail, "mlsender.net") {
		return sendEmailWithMailerSend(apiToken, senderEmail, recipientEmail, subject, body)
	} else if strings.Contains(senderEmail, "mailgun.org") {
		return sendEmailWithMailgun(apiToken, senderEmail, recipientEmail, subject, body)
	}
	return "Failed", "", fmt.Errorf("unsupported email provider")
}

func sendEmailWithMailerSend(apiToken, senderEmail, recipientEmail, subject, body string) (string, string, error) {
	url := "https://api.mailersend.com/v1/email"
	payload := strings.NewReader(fmt.Sprintf(`{
			"from": {"email": "%s"},
			"to": [{"email": "%s"}],
			"subject": "%s",
			"text": "%s"
	}`, senderEmail, recipientEmail, subject, body))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "Failed", "", err
	}
	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "Failed", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return "Failed", "", fmt.Errorf("failed to send email: %s", resp.Status)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	messageID := result["x-message-id"].(string)

	return "Delivered", messageID, nil
}

func sendEmailWithMailgun(apiToken, senderEmail, recipientEmail, subject, body string) (string, string, error) {
	parts := strings.Split(senderEmail, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid email format")
	}

	domainName := parts[1]

	reqUrl := "https://api.mailgun.net/v3/" + domainName + "/messages"
	data := &bytes.Buffer{}
	writer := multipart.NewWriter(data)

	// Add form fields
	fromFw, _ := writer.CreateFormField("from")
	_, err := io.Copy(fromFw, strings.NewReader(senderEmail))
	if err != nil {
		return "Failed", "", err
	}
	toFw, _ := writer.CreateFormField("to")
	_, err = io.Copy(toFw, strings.NewReader(recipientEmail))
	if err != nil {
		return "Failed", "", err
	}
	subjectFw, _ := writer.CreateFormField("subject")
	_, err = io.Copy(subjectFw, strings.NewReader(subject))
	if err != nil {
		return "Failed", "", err
	}
	htmlFw, _ := writer.CreateFormField("html")
	_, err = io.Copy(htmlFw, strings.NewReader(body))
	if err != nil {
		return "Failed", "", err
	}
	writer.Close()

	payload := bytes.NewReader(data.Bytes())
	req, err := http.NewRequest("POST", reqUrl, payload)
	if err != nil {
		return "Failed", "", err
	}
	req.SetBasicAuth("api", apiToken)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "Failed", "", err
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "Failed", "", err
	}

	if res.StatusCode != http.StatusOK {
		return "Failed", "", fmt.Errorf("failed to send email: %s", res.Status)
	}

	var result map[string]interface{}
	json.Unmarshal(bodyBytes, &result)
	messageID := result["id"].(string)

	return "Delivered", messageID, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmailReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&emailv1.Email{}).
		Complete(r)
}
