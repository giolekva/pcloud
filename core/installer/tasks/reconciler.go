package tasks

import (
	"context"
	"fmt"
	"net/http"
)

// TODO(gio): make reconciler address a flag
const baseURL = "http://fluxcd-reconciler.dodo-fluxcd-reconciler.svc.cluster.local"

type Reconciler interface {
	Reconcile(ctx context.Context, namespace, name string)
}

type SequentialReconciler struct {
	Reconcilers []Reconciler
}

func (r *SequentialReconciler) Reconcile(ctx context.Context, namespace, name string) {
	for _, rec := range r.Reconcilers {
		rec.Reconcile(ctx, namespace, name)
	}
}

type SourceGitReconciler struct{}

func (c SourceGitReconciler) Reconcile(ctx context.Context, namespace, name string) {
	addr := fmt.Sprintf("%s/source/git/%s/%s/reconcile", baseURL, namespace, name)
	http.Get(addr)
}

type KustomizationReconciler struct{}

func (c KustomizationReconciler) Reconcile(ctx context.Context, namespace, name string) {
	addr := fmt.Sprintf("%s/kustomization/%s/%s/reconcile", baseURL, namespace, name)
	http.Get(addr)
}

type namespaceNamePair struct {
	namespace string
	name      string
}

type FixedReconciler struct {
	namespace  string
	name       string
	reconciler Reconciler
}

func NewFixedReconciler(namespace, name string) *FixedReconciler {
	return &FixedReconciler{
		namespace,
		name,
		&SequentialReconciler{[]Reconciler{
			SourceGitReconciler{},
			// NOTE(gio): synchronizing git repository auto-syncs root kustomization as well
			// KustomizationReconciler{},
		}},
	}
}

func (r *FixedReconciler) Reconcile(ctx context.Context) {
	r.reconciler.Reconcile(ctx, r.namespace, r.name)
}
