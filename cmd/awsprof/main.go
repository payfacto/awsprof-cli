// Command awsprof picks an AWS profile to log in as. Living under
// cmd/awsprof so that `go install github.com/payfacto/awsprof-cli/cmd/awsprof`
// produces a binary named `awsprof` rather than `awsprof-cli`.
package main

import (
	"fmt"
	"os"

	"github.com/payfacto/awsprof-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
