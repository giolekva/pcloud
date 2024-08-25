package installer

import (
	"testing"
)

type testKeyGen struct{}

func (g testKeyGen) Generate(username string) (string, error) {
	return username, nil
}

func TestDeriveVPNAuthKey(t *testing.T) {
	schema := structSchema{
		"input",
		[]Field{
			Field{"username", basicSchema{"username", KindString, false, nil}},
			Field{"authKey", basicSchema{"authKey", KindVPNAuthKey, false, map[string]string{
				"usernameField": "username",
			}}},
		},
		false,
	}
	input := map[string]any{
		"username": "foo",
	}
	v, err := deriveValues(input, input, schema, nil, testKeyGen{})
	if err != nil {
		t.Fatal(err)
	}
	if key, ok := v["authKey"].(string); !ok || key != "foo" {
		t.Fatal(v)
	}
}
