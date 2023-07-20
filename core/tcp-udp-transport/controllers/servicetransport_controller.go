/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");time
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
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	transportv1 "github.com/giolekva/pcloud/api/v1"
	installer "github.com/giolekva/pcloud/core/installer"
)

// ServiceTransportReconciler reconciles a ServiceTransport object
type ServiceTransportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Signer ssh.Signer
}

//+kubebuilder:rbac:groups=transport.dodo.cloud,resources=servicetransports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=transport.dodo.cloud,resources=servicetransports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=transport.dodo.cloud,resources=servicetransports/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceTransport object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ServiceTransportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(req.String())

	resource := &transportv1.ServiceTransport{}
	if err := r.Get(context.Background(), client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, resource); err != nil {
		fmt.Println(err)
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	fmt.Printf("%+v\n", resource)
	if resource.Status.Port != 0 {
		return ctrl.Result{}, nil
	}

	ingressClassName := resource.Spec.IngressClassName
	labelSelector, err := labels.NewRequirement(
		"dodo.cloud/ingressClassName",
		selection.Equals,
		[]string{ingressClassName},
	)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	releases := unstructured.UnstructuredList{
		Object: map[string]interface{}{
			"kind":       "HelmRelease",
			"apiVersion": "helm.toolkit.fluxcd.io/v2beta1",
		},
		Items: []unstructured.Unstructured{},
	}
	if err := r.List(context.Background(), &releases, &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*labelSelector),
	}); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if len(releases.Items) == 0 {
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Ingress %s not found", ingressClassName)
	}
	if len(releases.Items) > 1 {
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Found more than one ingress %s", ingressClassName)
	}
	rel := releases.Items[0].Object
	repoAddr, repoPath, err := extractRepoInfo(rel)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	repo, err := cloneRepo(repoAddr, r.Signer)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	repoIO := installer.NewRepoIO(repo, r.Signer)
	data, err := repoIO.ReadYaml(repoPath)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	def, ok := data.(map[string]any)
	if !ok {
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Not map")
	}
	spec, ok := def["spec"].(map[string]any)
	if !ok {
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Not map")
	}
	values, ok := spec["values"].(map[string]any)
	if !ok {
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Not map")
	}
	protocol := strings.ToLower(resource.Spec.Protocol)
	var m map[string]any
	k, has := values[protocol]
	if !has {
		m = make(map[string]any)
		values[protocol] = m
	} else {
		m, ok = k.(map[string]any)
		if !ok {
			return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Not map")
		}
	}
	var sourcePort int
	if resource.Spec.SourcePort != 0 {
		sourcePort = resource.Spec.SourcePort
		if _, ok := m[strconv.Itoa(sourcePort)]; ok {
			return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("Source port %d is already taken", sourcePort)
		}
	} else {
		for {
			sourcePort = rand.Intn(65536-1000) + 1000
			if _, ok := m[strconv.Itoa(sourcePort)]; !ok {
				break
			}
		}
	}
	dest := fmt.Sprintf("%s/%s:%d", resource.ObjectMeta.Namespace, resource.Spec.Service, resource.Spec.Port)
	m[strconv.Itoa(sourcePort)] = dest
	if err := repoIO.WriteYaml(repoPath, def); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	if err := repoIO.CommitAndPush(fmt.Sprintf("%s transport for %s", protocol, dest)); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}
	resource.Status.Port = sourcePort
	if err := r.Status().Update(context.Background(), resource); err != nil {
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceTransportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&transportv1.ServiceTransport{}).
		Complete(r)
}

func extractRepoInfo(o map[string]any) (string, string, error) {
	metadata, ok := o["metadata"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("Not map")
	}
	annotations, ok := metadata["annotations"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("Not map")
	}
	repoAddr, ok := annotations["dodo.cloud/releaseSourceRepo"].(string)
	if !ok {
		return "", "", fmt.Errorf("Not string")
	}
	repoPath, ok := annotations["dodo.cloud/releasePath"].(string)
	if !ok {
		return "", "", fmt.Errorf("Not string")
	}
	return repoAddr, repoPath, nil
}

func cloneRepo(address string, signer ssh.Signer) (*git.Repository, error) {
	return git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:             address,
		Auth:            auth(signer),
		RemoteName:      "origin",
		InsecureSkipTLS: true,
	})
}

func auth(signer ssh.Signer) *gitssh.PublicKeys {
	return &gitssh.PublicKeys{
		Signer: signer,
		HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				// TODO(giolekva): verify server public key
				// fmt.Printf("## %s || %s -- \n", serverPubKey, ssh.MarshalAuthorizedKey(key))
				return nil
			},
		},
	}
}
