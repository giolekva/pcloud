package installer

import (
	"testing"
)

func TestFindPortFields(t *testing.T) {
	scm := structSchema{
		"a",
		[]Field{
			Field{"x", basicSchema{"x", KindString, false, nil}},
			Field{"y", basicSchema{"y", KindInt, false, nil}},
			Field{"z", basicSchema{"z", KindPort, false, nil}},
			Field{
				"w",
				structSchema{
					"w",
					[]Field{
						Field{"x", basicSchema{"x", KindString, false, nil}},
						Field{"y", basicSchema{"y", KindInt, false, nil}},
						Field{"z", basicSchema{"z", KindPort, false, nil}},
					},
					false,
				},
			},
		},
		false,
	}
	p := findPortFields(scm)
	if len(p) != 2 {
		t.Fatalf("expected two port fields, %v", p)
	}
	if p[0] != "z" || p[1] != "w.z" {
		t.Fatalf("expected 'z' and 'w.z' port fields, %v", p)
	}
}
