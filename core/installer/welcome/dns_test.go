package welcome

import (
	"strings"
	"testing"
)

const rec = `
t40.lekva.me.	3600	IN	DS	43870 13 2 9ADA4E046EC0473383035B7BDB6443B8D869A9C8B35D000B8038ABF3F3864621
ns1.t40.lekva.me. 10800 IN A 135.181.48.180
ns2.t40.lekva.me. 10800 IN A 65.108.39.172
t40.lekva.me. 10800 IN NS ns1.t40.lekva.me.
t40.lekva.me. 10800 IN NS ns2.t40.lekva.me.
`

func TestParse(t *testing.T) {
	zone := "lekva.me."
	recs, err := ParseRecords(zone, strings.Split(rec, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 5 {
		t.Fatalf("Expected 5 records, got %d", len(recs))
	}
}
