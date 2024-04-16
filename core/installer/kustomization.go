package installer

import (
	"bytes"
	"golang.org/x/exp/slices"
	"io"

	"sigs.k8s.io/yaml"
)

type Kustomization struct {
	ApiVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Resources  []string `json:"resources,omitempty"`
}

func NewKustomization() Kustomization {
	return Kustomization{
		ApiVersion: "kustomize.config.k8s.io/v1beta1",
		Kind:       "Kustomization",
		Resources:  []string{},
	}
}

func (k Kustomization) Write(w io.Writer) error {
	contents, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, bytes.NewReader(contents)); err != nil {
		return err
	}
	return nil
}

func (k *Kustomization) AddResources(names ...string) {
	for _, name := range names {
		if !slices.Contains(k.Resources, name) {
			k.Resources = append(k.Resources, name)
		}
	}
}

func (k *Kustomization) RemoveResources(names ...string) {
	for _, name := range names {
		for i, r := range k.Resources {
			if r == name {
				k.Resources = slices.Delete(k.Resources, i, i+1)
				break
			}
		}
	}
}
