package appmanager

type TriggerOn struct {
	Type  string `yaml:"type"`
	Event string `yaml:"event"`
}

type Trigger struct {
	Name      string    `yaml:"name"`
	TriggerOn TriggerOn `yaml:"triggerOn"`
	Action    string    `yaml:"action"`
}

type Triggers struct {
	Triggers []Trigger `yaml:"triggers"`
}
