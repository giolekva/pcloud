package installer

import (
	"crypto/rand"
	"fmt"
)

type SuffixGenerator interface {
	Generate() (string, error)
}

type emptySuffixGenerator struct {
}

func NewEmptySuffixGenerator() SuffixGenerator {
	return &emptySuffixGenerator{}
}

func (g *emptySuffixGenerator) Generate() (string, error) {
	return "", nil
}

type suffixGenerator struct {
	suffix string
}

func NewSuffixGenerator(suffix string) SuffixGenerator {
	return &suffixGenerator{suffix}
}

func (g *suffixGenerator) Generate() (string, error) {
	return g.suffix, nil
}

type fixedLengthRandomSuffixGenerator struct {
	len int
}

func NewFixedLengthRandomSuffixGenerator(len int) SuffixGenerator {
	return &fixedLengthRandomSuffixGenerator{len}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func (g *fixedLengthRandomSuffixGenerator) Generate() (string, error) {
	r := make([]byte, g.len)
	if _, err := rand.Read(r); err != nil {
		return "", err
	}
	ret := make([]rune, g.len)
	for i, v := range r {
		ret[i] += letters[v%26]
	}
	return fmt.Sprintf("-%s", string(ret)), nil
}

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
