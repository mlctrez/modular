package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSemVerTag_GreaterThan(t *testing.T) {
	req := require.New(t)

	req.True(SemVerTag{0, 0, 1}.GreaterThan(SemVerTag{0, 0, 0}))
	req.False(SemVerTag{0, 0, 0}.GreaterThan(SemVerTag{0, 0, 1}))
	req.True(SemVerTag{0, 1, 0}.GreaterThan(SemVerTag{0, 0, 99}))
	req.True(SemVerTag{1, 0, 0}.GreaterThan(SemVerTag{0, 99, 99}))

}
