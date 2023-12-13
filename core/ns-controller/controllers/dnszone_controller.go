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
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dodocloudv1 "github.com/giolekva/pcloud/core/ns-controller/api/v1"
)

// DNSZoneReconciler reconciles a DNSZone object
type DNSZoneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Store  ZoneStoreFactory
}

type DNSSecKey struct {
	Basename string `json:"basename,omitempty"`
	Key      []byte `json:"key,omitempty"`
	Private  []byte `json:"private,omitempty"`
	DS       []byte `json:"ds,omitempty"`
}

//+kubebuilder:rbac:groups=dodo.cloud.dodo.cloud,resources=dnszones,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dodo.cloud.dodo.cloud,resources=dnszones/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dodo.cloud.dodo.cloud,resources=dnszones/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DNSZone object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *DNSZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Store.Debug()
	defer func() {
		r.Store.Debug()
	}()
	logger := log.FromContext(ctx)
	logger.Info(req.String())

	resource := &dodocloudv1.DNSZone{}
	if err := r.Get(context.Background(), client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, resource); err != nil {
		if apierrors.IsGone(err) {
			fmt.Printf("GONE %s %s\n", req.Name, req.Namespace)
		} else {
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
	}
	if resource.Status.Ready {
		return ctrl.Result{}, nil
	}
	zoneConfig := ZoneConfig{
		Zone:        resource.Spec.Zone,
		PublicIPs:   resource.Spec.PublicIPs,
		PrivateIP:   resource.Spec.PrivateIP,
		Nameservers: resource.Spec.Nameservers,
	}
	if resource.Spec.DNSSec.Enabled {
		var secret corev1.Secret
		if err := r.Get(context.Background(), client.ObjectKey{
			Namespace: resource.Namespace, // NOTE(gio): configurable on resource level?
			Name:      resource.Spec.DNSSec.SecretName,
		}, &secret); err != nil {
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		basename, ok := secret.Data["basename"]
		if !ok {
			return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("basename not found")
		}
		key, ok := secret.Data["key"]
		if !ok {
			return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("key not found")
		}
		private, ok := secret.Data["private"]
		if !ok {
			return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("private not found")
		}
		ds, ok := secret.Data["ds"]
		if !ok {
			return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("ds not found")
		}
		zoneConfig.DNSSec = &DNSSecKey{
			Basename: string(basename),
			Key:      key,
			Private:  private,
			DS:       ds,
		}
	}
	zs, err := r.Store.Create(zoneConfig)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if err := zs.CreateConfigFile(); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	resource.Status.Ready = true
	if zoneConfig.DNSSec != nil {
		rrs := []string{string(zoneConfig.DNSSec.DS)}
		rrs = append(rrs, GenerateNSRecords(zoneConfig)...)
		resource.Status.RecordsToPublish = strings.Join(rrs, "\n")
	}
	if err := r.Status().Update(context.Background(), resource); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DNSZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dodocloudv1.DNSZone{}).
		Complete(r)
}
