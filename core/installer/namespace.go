package installer

import (
	"crypto/rand"
	"fmt"
)

type NamespaceGenerator interface {
	Generate(name string) (string, error)
}

type prefixGenerator struct {
	prefix string
}

func NewPrefixGenerator(prefix string) NamespaceGenerator {
	return &prefixGenerator{prefix}
}

func (g *prefixGenerator) Generate(name string) (string, error) {
	return g.prefix + name, nil
}

type randomSuffixGenerator struct {
	len int
}

func NewRandomSuffixGenerator(len int) NamespaceGenerator {
	return &randomSuffixGenerator{len}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func (g randomSuffixGenerator) Generate(name string) (string, error) {
	r := make([]byte, g.len)
	if _, err := rand.Read(r); err != nil {
		return "", err
	}
	ret := make([]rune, g.len)
	for i, v := range r {
		ret[i] += letters[v%26]
	}
	return fmt.Sprintf("%s-%s", name, string(ret)), nil
}

type combineGenerator struct {
	ns []NamespaceGenerator
}

func NewCombine(ns ...NamespaceGenerator) NamespaceGenerator {
	return &combineGenerator{ns}
}

func (g *combineGenerator) Generate(name string) (string, error) {
	cur := name
	var err error
	for _, i := range g.ns {
		cur, err = i.Generate(cur)
		if err != nil {
			return "", err
		}
	}
	return cur, nil
}
