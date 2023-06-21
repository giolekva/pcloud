/*
Copyright 2023.

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

package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	headscalev1 "github.com/giolekva/pcloud/api/v1"
)

type HeadscaleClient struct {
	baseUrl    url.URL
	httpClient *http.Client
}

func NewHeadscaleClient(baseUrl url.URL) *HeadscaleClient {
	return &HeadscaleClient{
		baseUrl,
		&http.Client{},
	}
}

func (c *HeadscaleClient) CreateUser(name string) error {
	reqAddr := c.baseUrl
	reqAddr.Path = "/user"
	req := &http.Request{
		Method: http.MethodPost,
		URL:    &reqAddr,
		Header: map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`
{
    "name": "%s"
}`, name))),
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Could not create user")
	}
	return nil
}

func (c *HeadscaleClient) CreateReusablePreAuthKey(user string) (string, error) {
	reqAddr := c.baseUrl
	reqAddr.Path = fmt.Sprintf("/user/%s/preauthkey", user)
	req := &http.Request{
		Method: http.MethodPost,
		URL:    &reqAddr,
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Could not create pre-authenticated key")
	}
	var contents bytes.Buffer
	if _, err := io.Copy(&contents, resp.Body); err != nil {
		return "", err
	}
	return contents.String(), nil
}

// HeadscaleUserReconciler reconciles a HeadscaleUser object
type HeadscaleUserReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Headscale *HeadscaleClient
}

//+kubebuilder:rbac:groups=headscale.dodo.cloud,resources=headscaleusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=headscale.dodo.cloud,resources=headscaleusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=headscale.dodo.cloud,resources=headscaleusers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HeadscaleUser object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *HeadscaleUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(req.String())

	resource := &headscalev1.HeadscaleUser{}
	if err := r.Get(context.Background(), client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, resource); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if resource.Status.Ready {
		return ctrl.Result{}, nil
	}
	baseAddr, err := url.Parse(resource.Spec.HeadscaleAddress)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	headscale := NewHeadscaleClient(*baseAddr)
	if err := headscale.CreateUser(resource.Spec.Name); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if resource.Spec.PreAuthKey.Enabled {
		key, err := headscale.CreateReusablePreAuthKey(resource.Spec.Name)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resource.Spec.PreAuthKey.SecretName,
				Namespace: req.Namespace,
			},
			StringData: map[string]string{
				"authkey": strings.TrimSpace(key),
			},
		}
		if err := r.Create(context.Background(), secret); err != nil {
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
	}
	resource.Status.Ready = true
	if err := r.Status().Update(context.Background(), resource); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HeadscaleUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&headscalev1.HeadscaleUser{}).
		Complete(r)
}
