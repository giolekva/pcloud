package appmanager

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type TriggerOn struct {
	Type  string `yaml:"type"`
	Event string `yaml:"event"`
}

type Trigger struct {
	Name      string    `yaml:"name"`
	TriggerOn TriggerOn `yaml:"triggerOn"`
	Template  string    `yaml:"template"`
}

type Triggers struct {
	Triggers []Trigger `yaml:"triggers"`
}

func TriggersFromYaml(str string) (*Triggers, error) {
	var s Triggers
	err := yaml.Unmarshal([]byte(str), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ReadTriggers(actionsFile string) (*Triggers, error) {
	f, err := os.Open(actionsFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return TriggersFromYaml(string(b))
}
