package appmanager

import (
	"log"
	"testing"
)

var tmpl = `
actions:
- name: DetectFaces
  triggerOn:
    type: Image
    event: NEW
  template: |
    kind: Pod
    apiVersion: v1
    metadata:
      name: detect-faces-{{ .Image.Id }}
    spec:
      containers:
      - name: detect-faces
        image: giolekva/face-detector:latest
        imagePullPolicy: Always
        command: ["python3", "main.py"]
        args: [{{ .PCloudApiAddr }}, {{ .ObjectStoreAddr }}, {{ .Image.Id }}]
      restartPolicy: Never`

func TestParse(t *testing.T) {
	a, err := ActionsFromYaml(tmpl)
	if err != nil {
		panic(err)
	}
	log.Print(a)
}
