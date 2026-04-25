// Command trailhead provides a tiny CLI: version, help, and a stub for `show` (line IDs
// resolve in the live MCP session via get_lines_by_id, not a separate process).
package main

import (
	"fmt"
	"os"

	"github.com/michalstefanec/trailhead/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "version", "-v", "--version":
		fmt.Println(version.Version)
	case "show":
		_, _ = fmt.Fprintln(os.Stderr, "trailhead: line_ids are session-scoped. Resolve evidence with the get_lines_by_id MCP tool in the same Trailhead process.")
		os.Exit(2)
	case "help", "-h", "--help":
		usage()
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Printf(`Trailhead — %s

Usage:
  trailhead help|version|show

Commands:
  version   print server/CLI build version
  show      line lookup is not stateful across processes; use get_lines_by_id in MCP
`, version.Version)
	fmt.Println()
}
