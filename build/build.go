package build

import "fmt"

var (
	Version string = "localbuild"
	Commit  string = ""
)

func trimRight(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[:n]
}

func AbsoluteVersion() string {
	if Commit == "" {
		return Version
	}

	return fmt.Sprintf("%s-%s", Version, trimRight(Commit, 8))
}
