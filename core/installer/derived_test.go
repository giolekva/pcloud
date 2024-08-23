package installer

import (
	"net"
	"testing"
)

type testKeyGen struct{}

func (g testKeyGen) GenerateAuthKey(username string) (string, error) {
	return username, nil
}

func (g testKeyGen) ExpireKey(username, key string) error {
	return nil
}

func (g testKeyGen) ExpireNode(username, node string) error {
	return nil
}

func (g testKeyGen) RemoveNode(username, node string) error {
	return nil
}

func (g testKeyGen) GetNodeIP(username, node string) (net.IP, error) {
	return nil, nil
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
	v, err := deriveValues(input, input, schema, nil, nil, testKeyGen{})
	if err != nil {
		t.Fatal(err)
	}
	if key, ok := v["authKey"].(string); !ok || key != "foo" {
		t.Fatal(v)
	}
}

func TestDeriveVPNAuthKeyDisabled(t *testing.T) {
	schema := structSchema{
		"input",
		[]Field{
			Field{"username", basicSchema{"username", KindString, false, nil}},
			Field{"enabled", basicSchema{"enabled", KindBoolean, false, nil}},
			Field{"authKey", basicSchema{"authKey", KindVPNAuthKey, false, map[string]string{
				"usernameField": "username",
				"enabledField":  "enabled",
			}}},
		},
		false,
	}
	input := map[string]any{
		"username": "foo",
		"enabled":  false,
	}
	v, err := deriveValues(input, input, schema, nil, nil, testKeyGen{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v["authKey"].(string); ok {
		t.Fatal(v)
	}
}

func TestDeriveVPNAuthKeyEnabledExplicitly(t *testing.T) {
	schema := structSchema{
		"input",
		[]Field{
			Field{"username", basicSchema{"username", KindString, false, nil}},
			Field{"enabled", basicSchema{"enabled", KindBoolean, false, nil}},
			Field{"authKey", basicSchema{"authKey", KindVPNAuthKey, false, map[string]string{
				"usernameField": "username",
				"enabledField":  "enabled",
			}}},
		},
		false,
	}
	input := map[string]any{
		"username": "foo",
		"enabled":  true,
	}
	v, err := deriveValues(input, input, schema, nil, nil, testKeyGen{})
	if err != nil {
		t.Fatal(err)
	}
	if key, ok := v["authKey"].(string); !ok || key != "foo" {
		t.Fatal(v)
	}
}
