package tasks

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Reconciler interface {
	Reconcile(ctx context.Context)
}

type fluxcdReconciler struct {
	resources []string
}

func NewFluxcdReconciler(addr, name string) Reconciler {
	return fluxcdReconciler{
		resources: []string{
			fmt.Sprintf("%s/source/git/%s/%s/reconcile", addr, name, name),
			fmt.Sprintf("%s/kustomization/%s/%s/reconcile", addr, name, name),
		},
	}
}

func (r fluxcdReconciler) Reconcile(ctx context.Context) {
	for {
		select {
		case <-time.After(3 * time.Second):
			for _, res := range r.resources {
				http.Get(res)
			}
		case <-ctx.Done():
			return
		}
	}
}
