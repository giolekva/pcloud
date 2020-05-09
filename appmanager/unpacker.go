package appmanager

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

type Unpacker interface {
	Unpack(archive string,
		namespace string,
		values map[string]string) (map[string][]string, error)
}

type helmUnpacker struct {
	helmBin string
}

func NewHelmUnpacker(helmBin string) Unpacker {
	return &helmUnpacker{helmBin}
}

func (h *helmUnpacker) Unpack(
	archive string,
	namespace string,
	values map[string]string) (map[string][]string, error) {
	cmd := h.generateHelmInstallCmd(archive, namespace, values)
	glog.Info(cmd.String())
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, errors.New(stderr.String())
	}
	return extractTemplates(stdout.String())
}

func (h *helmUnpacker) generateHelmInstallCmd(
	archive string,
	namespace string,
	values map[string]string) *exec.Cmd {
	cmd := exec.Command(h.helmBin)
	cmd.Args = append(cmd.Args, "template")
	cmd.Args = append(cmd.Args, fmt.Sprintf("--namespace=%s", namespace))
	cmd.Args = append(cmd.Args, "--generate-name")
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s", archive))
	// TODO(giolekva): validate values
	for key, value := range values {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--set=%s=%s", key, value))
	}
	return cmd
}

func extractTemplates(bundle string) (map[string][]string, error) {
	items := strings.Split(bundle, "---")
	temps := make(map[string][]string)
	for _, item := range items {
		if len(item) == 0 {
			continue
		}
		tmp := strings.SplitN(item, "\n", 3)
		if len(tmp) != 3 {
			return nil, fmt.Errorf("Got invalid template: %s", item)
		}
		source := tmp[1]
		glog.Info(source)
		// if !strings.HasPrefix(source, "\n# Source: ") {
		// 	return nil, fmt.Errorf("Got invalid source: %s", item)
		// }
		sourceItems := strings.Split(source, "/")
		glog.Info(sourceItems)
		if len(sourceItems) != 3 {
			return nil, fmt.Errorf("Got invalid source: %s", item)
		}
		path := sourceItems[1] + "/" + sourceItems[2]
		if _, ok := temps[path]; !ok {
			temps[path] = make([]string, 1)
		}
		temps[path] = append(temps[path], tmp[2])
	}
	return temps, nil
}
