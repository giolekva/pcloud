package appmanager

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

type Chart struct {
	Name string `yaml:"name"`
}

type HelmChart struct {
	Chart
	chartDir string
	Schema   *Schema
	Yamls    []string
}

func HelmChartFromDir(chartDir string) (*HelmChart, error) {
	var chart HelmChart
	chart.chartDir = chartDir
	c, err := ReadChart(path.Join(chartDir, "Chart.yaml"))
	if err != nil {
		return nil, err
	}
	chart.Chart = *c
	schema, err := ReadSchema(path.Join(chartDir, "Schema.yaml"))
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}
	chart.Schema = schema
	return &chart, nil
}

func HelmChartFromTar(chartTar string) (*HelmChart, error) {
	if !strings.HasSuffix(chartTar, ".tar.gz") {
		return nil, errors.New("Expected .tar.gz file")
	}
	dir := filepath.Dir(chartTar)
	archive := filepath.Base(chartTar)
	if err := syscall.Chdir(dir); err != nil {
		return nil, err
	}
	cmd := exec.Command("tar", "-xvf", archive)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		glog.Info("-----")
		return nil, errors.New(stderr.String())
	}
	glog.Info(stdout.String())
	glog.Info(dir)
	return HelmChartFromDir(dir)
}

func (h *HelmChart) Install(
	helmBin string,
	values map[string]string) error {
	namespace := fmt.Sprintf("app-%s", h.Chart.Name)
	cmd := generateHelmInstallCmd(helmBin, h.chartDir, namespace, values)
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

func generateHelmInstallCmd(
	helmBin string,
	archive string,
	namespace string,
	values map[string]string) *exec.Cmd {
	cmd := exec.Command(helmBin)
	cmd.Args = append(cmd.Args, "install")
	cmd.Args = append(cmd.Args, fmt.Sprintf("--namespace=%s", namespace))
	cmd.Args = append(cmd.Args, "--generate-name")
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s", archive))
	// TODO(giolekva): validate values
	for key, value := range values {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--set=%s=%s", key, value))
	}
	return cmd
}

func ChartFromYaml(str string) (*Chart, error) {
	var s Chart
	err := yaml.Unmarshal([]byte(str), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ReadChart(chartFile string) (*Chart, error) {
	f, err := os.Open(chartFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return ChartFromYaml(string(b))
}
