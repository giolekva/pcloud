package installer

import (
	"bytes"
	"golang.org/x/exp/slices"
	"io"
	"io/ioutil"

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

func ReadKustomization(r io.Reader) (*Kustomization, error) {
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var ret Kustomization
	if err = yaml.UnmarshalStrict(contents, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
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
