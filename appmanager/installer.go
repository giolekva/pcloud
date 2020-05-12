package appmanager

import (
	"fmt"
	"net/http"
	"strings"
)

func InstallSchema(schema *Schema, apiAddr string) error {
	if schema == nil || len(schema.Schema) == 0 {
		return nil
	}
	resp, err := http.Post(apiAddr, "application/text", strings.NewReader(schema.Schema))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed request with status code: %d", resp.StatusCode)
	}
	return nil
}
