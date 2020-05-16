package appmanager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

type Chart struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type HelmChart struct {
	Chart
	Dir       string
	Namespace string
	Schema    Schema
	Triggers  Triggers
	Actions   Actions
	Init      Init
	Yamls     []string
}

func HelmChartFromDir(dir string) (*HelmChart, error) {
	var chart HelmChart
	chart.Dir = dir
	err := FromYamlFile(path.Join(dir, "Chart.yaml"), &chart.Chart)
	if err != nil {
		return nil, err
	}
	return &chart, nil
}

func (chart *HelmChart) Render(
	helmBin string,
	values map[string]string) error {
	chart.Namespace = fmt.Sprintf("app-%s", chart.Name)
	renderDir := path.Join(chart.Dir, "__render")
	if err := chart.renderTemplates(helmBin, values, renderDir); err != nil {
		return err
	}
	if err := os.RemoveAll(path.Join(chart.Dir, "templates")); err != nil {
		return err
	}
	if err := os.Rename(
		path.Join(renderDir, chart.Name, "templates"),
		path.Join(chart.Dir, "templates")); err != nil {
		return err
	}
	pcloudDir := path.Join(chart.Dir, "templates/pcloud")
	err := FromYamlFile(path.Join(pcloudDir, "Schema.yaml"), &chart.Schema)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = FromYamlFile(path.Join(pcloudDir, "Triggers.yaml"), &chart.Triggers)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = FromYamlFile(path.Join(pcloudDir, "Actions.yaml"), &chart.Actions)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	err = FromYamlFile(path.Join(pcloudDir, "Init.yaml"), &chart.Init)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.RemoveAll(pcloudDir); err != nil {
		return err
	}
	return nil
}

func HelmChartFromTar(chartTar string) (*HelmChart, error) {
	if !strings.HasSuffix(chartTar, ".tar.gz") {
		return nil, errors.New("Expected .tar.gz file")
	}
	dir := filepath.Dir(chartTar)
	archive := filepath.Base(chartTar)
	if err := os.Chdir(dir); err != nil {
		return nil, err
	}
	cmd := exec.Command("tar", "-xvf", archive)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.New(stderr.String())
	}
	glog.Info(stdout.String())
	glog.Info(dir)
	return HelmChartFromDir(dir)
}

func (chart *HelmChart) Install(
	helmBin string) error {
	cmd := exec.Command(helmBin)
	cmd.Args = append(cmd.Args, "install")
	cmd.Args = append(cmd.Args, fmt.Sprintf("--namespace=%s", chart.Namespace))
	cmd.Args = append(cmd.Args, chart.Name)
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s", chart.Dir))
	return runCmd(cmd)
}

func (chart *HelmChart) renderTemplates(
	helmBin string,
	values map[string]string,
	outputDir string) error {
	cmd := exec.Command(helmBin)
	cmd.Args = append(cmd.Args, "template")
	cmd.Args = append(cmd.Args, fmt.Sprintf("--output-dir=%s", outputDir))
	cmd.Args = append(cmd.Args, fmt.Sprintf("--namespace=%s", chart.Namespace))
	cmd.Args = append(cmd.Args, chart.Name)
	cmd.Args = append(cmd.Args, chart.Dir)
	// TODO(giolekva): validate values
	for key, value := range values {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--set=%s=%s", key, value))
	}
	return runCmd(cmd)
}

func runCmd(cmd *exec.Cmd) error {
	glog.Info(cmd.String())
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(stderr.String())
	}
	glog.Info(stdout.String())
	return nil
}
