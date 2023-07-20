package main

import (
	_ "embed"
	"fmt"
	"net/netip"
	"os"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"

	"github.com/giolekva/pcloud/core/installer"
)

var bootstrapFlags struct {
	envName                   string
	publicIP                  string
	chartsDir                 string
	adminPubKey               string
	storageDir                string
	volumeDefaultReplicaCount int
	fromIP                    string
	toIP                      string
}

func bootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "bootstrap",
		RunE: bootstrapCmdRun,
	}
	cmd.Flags().StringVar(
		&bootstrapFlags.envName,
		"env-name",
		"pcloud",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.envName,
		"public-ip",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.chartsDir,
		"charts-dir",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.adminPubKey,
		"admin-pub-key",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.storageDir,
		"storage-dir",
		"",
		"",
	)
	cmd.Flags().IntVar(
		&bootstrapFlags.volumeDefaultReplicaCount,
		"volume-default-replica-count",
		3,
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.fromIP,
		"from-ip",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&bootstrapFlags.toIP,
		"to-ip",
		"",
		"",
	)
	return cmd
}

func bootstrapCmdRun(cmd *cobra.Command, args []string) error {
	adminPubKey, err := os.ReadFile(bootstrapFlags.adminPubKey)
	if err != nil {
		return err
	}
	nsCreator, err := newNSCreator()
	if err != nil {
		return err
	}
	serviceIPs, err := newServiceIPs(bootstrapFlags.fromIP, bootstrapFlags.toIP)
	if err != nil {
		return err
	}
	envConfig := installer.EnvConfig{
		Name:                      bootstrapFlags.envName,
		PublicIP:                  bootstrapFlags.publicIP,
		NamespacePrefix:           fmt.Sprintf("%s-", bootstrapFlags.envName),
		StorageDir:                bootstrapFlags.storageDir,
		VolumeDefaultReplicaCount: bootstrapFlags.volumeDefaultReplicaCount,
		AdminPublicKey:            adminPubKey,
		ServiceIPs:                serviceIPs,
	}
	b := installer.NewBootstrapper(
		installer.NewFSChartLoader(bootstrapFlags.chartsDir),
		nsCreator,
		actionConfigFactory{rootFlags.kubeConfig},
	)
	return b.Run(envConfig)
}

type actionConfigFactory struct {
	kubeConfigPath string
}

func (f actionConfigFactory) New(namespace string) (*action.Configuration, error) {
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

func newServiceIPs(from, to string) (installer.EnvServiceIPs, error) {
	f, err := netip.ParseAddr(from)
	if err != nil {
		return installer.EnvServiceIPs{}, err
	}
	t, err := netip.ParseAddr(to)
	if err != nil {
		return installer.EnvServiceIPs{}, err
	}
	configRepo := f
	ingressPublic := configRepo.Next()
	restFrom := ingressPublic.Next()
	return installer.EnvServiceIPs{
		ConfigRepo:    configRepo,
		IngressPublic: ingressPublic,
		From:          restFrom,
		To:            t,
	}, nil
}
