package build

import (
	"fmt"
	"strings"
)

var (
	Version string = "localbuild"
	Commit  string = ""
	Build   string = ""
)

func trim(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

func AbsoluteVersion() string {
	var s strings.Builder

	fmt.Fprint(&s, Version)

	if Commit != "" {
		fmt.Fprintf(&s, "-%s", trim(Commit, 8))
	}

	if Build != "" {
		fmt.Fprintf(&s, "-%s", Build)
	}

	return s.String()
}
