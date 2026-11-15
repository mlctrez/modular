package impl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/config"
)

// SemVerTag is a vMAJOR.MINOR.REVISION tag. Parsed is false for the zero value.
type SemVerTag struct {
	Major    int
	Minor    int
	Revision int
	Parsed   bool
}

func (sv SemVerTag) RefSpec() config.RefSpec {
	return config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", sv, sv))
}

func (sv SemVerTag) String() string {
	return fmt.Sprintf("v%d.%d.%d", sv.Major, sv.Minor, sv.Revision)
}

func (sv SemVerTag) GreaterThan(other SemVerTag) bool {
	if sv.Major != other.Major {
		return sv.Major > other.Major
	}
	if sv.Minor != other.Minor {
		return sv.Minor > other.Minor
	}
	return sv.Revision > other.Revision
}

// NextRevision returns the tag with the revision incremented.
func (sv SemVerTag) NextRevision() SemVerTag {
	sv.Revision++
	sv.Parsed = true
	return sv
}

// ParseVersionString parses "1.2.3" (no v prefix).
func ParseVersionString(in string) (SemVerTag, error) {
	parts := strings.Split(in, ".")
	if len(parts) != 3 {
		return SemVerTag{}, fmt.Errorf("not in three parts")
	}
	var nums [3]int
	for i, name := range []string{"major", "minor", "revision"} {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 {
			return SemVerTag{}, fmt.Errorf("%s not int", name)
		}
		nums[i] = n
	}
	return SemVerTag{Major: nums[0], Minor: nums[1], Revision: nums[2], Parsed: true}, nil
}

// Highest returns the greatest vX.Y.Z among the tag names, ignoring everything else.
func Highest(names []string) (latest SemVerTag) {
	for _, name := range names {
		if !strings.HasPrefix(name, "v") {
			continue
		}
		if tag, err := ParseVersionString(strings.TrimPrefix(name, "v")); err == nil && tag.GreaterThan(latest) {
			latest = tag
		}
	}
	return
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
