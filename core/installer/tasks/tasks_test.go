package tasks

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLeaf(t *testing.T) {
	l := newLeafTask("leaf", func() error {
		return nil
	})
	done := make(chan error)
	l.OnDone(func(err error) {
		done <- err
	})
	go l.Start()
	err := <-done
	if err != nil {
		t.Fatalf("Expected nil, got %s", err.Error())
	}
}

func TestSequentialSuccess(t *testing.T) {
	one := newLeafTask("one", func() error {
		return nil
	})
	two := newLeafTask("two", func() error {
		return nil
	})
	l := newSequentialParentTask("parent", true, &one, &two)
	done := make(chan error)
	l.OnDone(func(err error) {
		done <- err
	})
	go l.Start()
	err := <-done
	if err != nil {
		t.Fatalf("Expected nil, got %s", err.Error())
	}
}

func TestSequentialFailsFirst(t *testing.T) {
	one := newLeafTask("one", func() error {
		return fmt.Errorf("one")
	})
	two := newLeafTask("two", func() error {
		return nil
	})
	l := newSequentialParentTask("parent", true, &one, &two)
	done := make(chan error)
	l.OnDone(func(err error) {
		done <- err
	})
	go l.Start()
	err := <-done
	if err == nil || err.Error() != "one" {
		t.Fatalf("Expected one, got %s", err)
	}
}

func TestSequentialFailsSecond(t *testing.T) {
	one := newLeafTask("one", func() error {
		fmt.Println("one")
		return nil
	})
	two := newLeafTask("two", func() error {
		fmt.Println("two")
		return fmt.Errorf("two")
	})
	l := newSequentialParentTask("parent", true, &one, &two)
	done := make(chan error)
	l.OnDone(func(err error) {
		done <- err
	})
	go l.Start()
	err := <-done
	if err == nil || err.Error() != "two" {
		t.Fatalf("Expected two, got %s", err)
	}
}

func TestConcurrentTaskSucceeds_WaitsForAllChildren(t *testing.T) {
	cnt := 0
	var m sync.Mutex
	one := newLeafTask("one", func() error {
		m.Lock()
		defer m.Unlock()
		cnt++
		return nil
	})
	two := newLeafTask("two", func() error {
		time.Sleep(1 * time.Second)
		m.Lock()
		defer m.Unlock()
		cnt++
		return nil
	})
	l := newConcurrentParentTask("parent", true, &one, &two)
	done := make(chan error)
	l.OnDone(func(err error) {
		done <- err
	})
	go l.Start()
	err := <-done
	if err != nil {
		t.Fatalf("Expected nil, got %s", err.Error())
	}
	if cnt != 2 {
		t.Fatalf("Expected 2, got %d", cnt)
	}
}
