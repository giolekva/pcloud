package installer

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
)

type ActionConfigFactory struct {
	kubeConfigPath string
}

func NewActionConfigFactory(kubeConfigPath string) ActionConfigFactory {
	return ActionConfigFactory{kubeConfigPath}
}

func (f ActionConfigFactory) New(namespace string) (*action.Configuration, error) {
	config := new(action.Configuration)
	if err := config.Init(
		kube.GetConfig(f.kubeConfigPath, "", namespace),
		namespace,
		"",
		func(fmtString string, args ...any) {
			fmt.Printf(fmtString, args...)
			fmt.Println()
		},
	); err != nil {
		return nil, err
	}
	return config, nil
}
