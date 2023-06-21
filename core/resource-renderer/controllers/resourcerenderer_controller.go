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
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	dodocloudv1 "github.com/giolekva/pcloud/api/v1"
)

// ResourceRendererReconciler reconciles a ResourceRenderer object
type ResourceRendererReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dodo.cloud.dodo.cloud,resources=resourcerenderers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dodo.cloud.dodo.cloud,resources=resourcerenderers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dodo.cloud.dodo.cloud,resources=resourcerenderers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ResourceRenderer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ResourceRendererReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(req.String())

	resource := &dodocloudv1.ResourceRenderer{}
	if err := r.Get(context.Background(), client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, resource); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if resource.Status.Ready {
		return ctrl.Result{}, nil
	}
	secret := &corev1.Secret{}
	ns := resource.Spec.SecretNamespace
	if len(ns) == 0 {
		ns = req.Namespace
	}
	if err := r.Get(context.Background(), client.ObjectKey{
		Namespace: ns,
		Name:      resource.Spec.SecretName,
	}, secret); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	data := make(map[string]string)
	for key, value := range secret.Data {
		data[key] = string(value)
	}
	tmpl, err := template.New("resource").Parse(resource.Spec.ResourceTemplate)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	config := &corev1.ConfigMap{}
	if err := yaml.Unmarshal(rendered.Bytes(), config); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if err := r.Create(context.Background(), config); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	resource.Status.Ready = true
	if err := r.Status().Update(context.Background(), resource); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceRendererReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dodocloudv1.ResourceRenderer{}).
		Complete(r)
}
