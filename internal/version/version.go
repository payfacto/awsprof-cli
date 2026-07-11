// Package version holds the build version, injected via -ldflags at build time.
package version

// Version is the CLI version. It defaults to "dev" for plain `go build` and is
// overridden at release time via -ldflags -X.
var Version = "dev"
