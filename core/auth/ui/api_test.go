package main

import (
	"testing"
)

func TestPasswordInvalid(t *testing.T) {
	errs := validatePassword("foobar")
	if len(errs) != 2 {
		t.Fatal(errs)
	}
}

func TestPasswordValid(t *testing.T) {
	errs := validatePassword("foBa2r-gdkjS1-SA0120")
	if len(errs) != 0 {
		t.Fatal(errs)
	}
}
