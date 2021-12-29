package main

import (
	"embed"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var config = flag.String("config", "", "Path to config file")
var outputDir = flag.String("output-dir", "", "Path to the output directory")

//go:embed values-tmpl
var valuesTmpls embed.FS

var rootCmd *cobra.Command

var installFlags struct {
	config    string
	appName   string
	outputDir string
}

func init() {
	rootCmd = &cobra.Command{
		Use: "pcloud",
	}
	installCmd := &cobra.Command{
		Use:  "install",
		RunE: installCmdRun,
	}
	installCmd.Flags().StringVar(
		&installFlags.config,
		"config",
		"",
		"",
	)
	installCmd.Flags().StringVar(
		&installFlags.appName,
		"app",
		"",
		"",
	)
	installCmd.Flags().StringVar(
		&installFlags.outputDir,
		"output-dir",
		"",
		"",
	)
	rootCmd.AddCommand(installCmd)
}

func readConfig(config string) (Config, error) {
	var cfg Config
	inp, err := ioutil.ReadFile(config)
	if err != nil {
		return cfg, err
	}
	err = yaml.UnmarshalStrict(inp, &cfg)
	return cfg, err
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	cfg, err := readConfig(installFlags.config)
	if err != nil {
		return err
	}
	tmpls, err := template.ParseFS(valuesTmpls, "values-tmpl/*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	apps := []App{
		CreateAppIngressPrivate(tmpls),
		CreateAppCoreAuth(tmpls),
		CreateAppVaultwarden(tmpls),
		CreateAppMatrix(tmpls),
		CreateAppPihole(tmpls),
		CreateAppMaddy(tmpls),
	}
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
