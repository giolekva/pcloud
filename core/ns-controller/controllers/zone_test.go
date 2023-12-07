package controllers

import (
	"strings"
	"testing"

	"os"
)

const sample = `
example.com.   IN SOA ns1.example.com. hostmaster.example.com. 2015082541 7200 3600 1209600 3600
ns1.example.com. 10800 IN A 10.1.0.1
ns2.example.com. 10800 IN A 10.1.0.2
@.example.com. 10800 IN A 10.1.0.1
@.example.com. 10800 IN A 10.1.0.2
*.example.com. 10800 IN CNAME example.com.
p.example.com. 10800 IN CNAME example.com.
*.p.example.com. 10800 IN A 10.0.0.1
`

func TestRead(t *testing.T) {
	z, err := NewZoneFile(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	z.CreateOrReplaceTxtRecord("foo.example.com.", "bar")
	z.DeleteTxtRecord("foo.example.com.", "bar")
	if err := z.Write(os.Stdout); err != nil {
		t.Fatal(err)
	}
}
