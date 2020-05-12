package appmanager

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Schema struct {
	Schema string `yaml:"schema"`
}

func SchemaFromYaml(str string) (*Schema, error) {
	var s Schema
	err := yaml.Unmarshal([]byte(str), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ReadSchema(schemaFile string) (*Schema, error) {
	f, err := os.Open(schemaFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return SchemaFromYaml(string(b))
}
