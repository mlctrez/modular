package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSemVerTag_GreaterThan(t *testing.T) {
	req := require.New(t)

	p := func(in string) SemVerTag {
		tag, err := ParseVersionString(in)
		req.Nil(err)
		return tag
	}

	req.True(p("0.0.1").GreaterThan(p("0.0.0")))
	req.False(p("0.0.0").GreaterThan(p("0.0.1")))
	req.True(p("0.1.0").GreaterThan(p("0.0.99")))
	req.True(p("1.0.0").GreaterThan(p("0.99.99")))
	req.True(p("0.10.0").GreaterThan(p("0.5.99")))
	req.False(p("0.5.99").GreaterThan(p("0.10.0")))
}
