package impl

import (
	"errors"
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

func TestParseVersionString(t *testing.T) {
	req := require.New(t)
	tag, err := ParseVersionString("1.22.3")
	req.NoError(err)
	req.Equal(SemVerTag{Major: 1, Minor: 22, Revision: 3, Parsed: true}, tag)
	req.Equal("v1.22.3", tag.String())
	req.Equal("refs/tags/v1.22.3:refs/tags/v1.22.3", string(tag.RefSpec()))
	req.Equal("v1.22.4", tag.NextRevision().String())

	for in, want := range map[string]string{
		"1.2": "three parts", "a.2.3": "major", "1.b.3": "minor", "1.2.c": "revision", "1.-2.3": "minor", "": "three parts",
	} {
		tag, err = ParseVersionString(in)
		req.ErrorContains(err, want, in)
		req.False(tag.Parsed)
	}
}

func TestHighest(t *testing.T) {
	req := require.New(t)
	req.Equal("v0.10.0", Highest([]string{"v0.9.9", "release", "v0.10.0", "1.0.0", "v0.2.0", "vx.1.2"}).String())
	req.False(Highest(nil).Parsed)
	req.True(Highest([]string{"v0.0.1"}).Parsed)
	req.True(contains([]string{"a", "b"}, "b"))
	req.False(contains(nil, "b"))
}

func TestParseArgs(t *testing.T) {
	req := require.New(t)

	r, err := ParseArgs([]string{"bump", "a message"})
	req.NoError(err)
	req.True(r.Bump)
	req.Equal("a message", r.Message)
	req.False(r.Tag.Parsed)

	r, err = ParseArgs([]string{"msg", "1.2.3"})
	req.NoError(err)
	req.False(r.Bump)
	req.Equal("v1.2.3", r.Tag.String())
	req.Equal("msg", r.Message)

	r, err = ParseArgs([]string{"just a message"})
	req.NoError(err)
	req.False(r.Tag.Parsed)

	for _, bad := range [][]string{
		{"bump"}, {"bump", "1.0.0", "m"}, {"1.0.0", "bump", "m"}, {"bump", "bump", "m"}, {"1.0.0", "2.0.0", "m"},
		{"v1.0.0", "m"}, {"one", "two"}, {" "}, {},
	} {
		_, err = ParseArgs(bad)
		var ue *UsageError
		req.True(errors.As(err, &ue), "%v", bad)
	}
}
