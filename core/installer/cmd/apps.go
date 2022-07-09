package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/giolekva/pcloud/core/installer"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var installFlags struct {
	config    string
	appName   string
	outputDir string
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "install",
		RunE: installCmdRun,
	}
	cmd.Flags().StringVar(
		&installFlags.config,
		"config",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&installFlags.appName,
		"app",
		"",
		"",
	)
	cmd.Flags().StringVar(
		&installFlags.outputDir,
		"output-dir",
		"",
		"",
	)
	return cmd
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	cfg, err := readConfig(installFlags.config)
	if err != nil {
		return err
	}
	apps := installer.CreateAllApps()
	for _, a := range apps {
		if a.Name == installFlags.appName {
			for _, t := range a.Templates {
				out, err := os.Create(filepath.Join(installFlags.outputDir, t.Name()))
				if err != nil {
					return err
				}
				defer out.Close()
				if err := t.Execute(out, cfg); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

func readConfig(config string) (installer.Config, error) {
	var cfg installer.Config
	inp, err := ioutil.ReadFile(config)
	if err != nil {
		return cfg, err
	}
	err = yaml.UnmarshalStrict(inp, &cfg)
	return cfg, err
}
