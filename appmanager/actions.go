package appmanager

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Action struct {
	Name     string `yaml:"name"`
	Template string `yaml:"template"`
}

type Actions struct {
	Actions []Action `yaml:"actions"`
}

func FromYaml(str string, out interface{}) error {
	return yaml.Unmarshal([]byte(str), out)
}

func FromYamlFile(actionsFile string, out interface{}) error {
	f, err := os.Open(actionsFile)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return FromYaml(string(b), out)
}
