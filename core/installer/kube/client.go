package kube

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

type KubeConfigOpts struct {
	KubeConfig     string
	KubeConfigPath string
}

func NewKubeClient(opts KubeConfigOpts) (*kubernetes.Clientset, error) {
	if opts.KubeConfig != "" && opts.KubeConfigPath != "" {
		return nil, fmt.Errorf("both path and config can not be defined")
	}
	if opts.KubeConfig != "" {
		var cfg clientcmdapi.Config
		decoded, _, err := clientcmdlatest.Codec.Decode([]byte(opts.KubeConfig), &schema.GroupVersionKind{Version: clientcmdlatest.Version, Kind: "Config"}, &cfg)
		if err != nil {
			return nil, err
		}
		getter := func() (*clientcmdapi.Config, error) {
			return decoded.(*clientcmdapi.Config), nil
		}
		config, err := clientcmd.BuildConfigFromKubeconfigGetter("", getter)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}
	if opts.KubeConfigPath == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfigPath)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}
}
