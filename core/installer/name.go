package installer

import (
	"crypto/rand"
)

type NameGenerator interface {
	Generate() (string, error)
}

type fixedLengthRandomNameGenerator struct {
	len int
}

func NewFixedLengthRandomNameGenerator(len int) NameGenerator {
	return &fixedLengthRandomNameGenerator{len}
}

func (g *fixedLengthRandomNameGenerator) Generate() (string, error) {
	r := make([]byte, g.len)
	if _, err := rand.Read(r); err != nil {
		return "", err
	}
	ret := make([]rune, g.len)
	for i, v := range r {
		ret[i] += letters[v%26]
	}
	return string(ret), nil
}
