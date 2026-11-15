package impl

import (
	"fmt"
	"strings"
)

// UsageError is a mistake in the command line arguments.
type UsageError struct{ msg string }

func (e *UsageError) Error() string { return e.msg }

func usagef(format string, args ...any) error { return &UsageError{fmt.Sprintf(format, args...)} }

// Request is what the command line asked for.
type Request struct {
	Message string
	Bump    bool
	Tag     SemVerTag // explicit version, when Parsed
}

// ParseArgs splits the arguments into the commit message and the optional tag request.
func ParseArgs(args []string) (req Request, err error) {
	for _, arg := range args {
		t, perr := ParseVersionString(strings.TrimPrefix(arg, "v"))
		switch {
		case perr == nil && strings.HasPrefix(arg, "v"):
			return req, usagef("version %q: omit the v prefix, it is added automatically", arg)
		case arg == "bump" || perr == nil:
			if req.Bump || req.Tag.Parsed {
				return req, usagef("give only one of bump or a version")
			}
			req.Bump, req.Tag = arg == "bump", t
		case req.Message != "":
			return req, usagef("more than one commit message given, quote the message")
		default:
			req.Message = arg
		}
	}
	if strings.TrimSpace(req.Message) == "" {
		return req, usagef("a commit message is required")
	}
	return req, nil
}
