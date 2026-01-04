package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jonelmawirat/toon"
)

func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, in io.Reader, out io.Writer, errOut io.Writer) int {
	cmd := "fmt"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		args = args[1:]
	}

	fs := flag.NewFlagSet("toon "+cmd, flag.ContinueOnError)
	fs.SetOutput(errOut)

	strict := fs.Bool("strict", true, "")
	expand := fs.String("expand-paths", "off", "")
	indent := fs.Int("indent", 2, "")
	docDelim := fs.String("doc-delim", ",", "")
	arrDelim := fs.String("array-delim", ",", "")
	fold := fs.String("fold", "off", "")
	flattenDepth := fs.Int("flatten-depth", int(^uint(0)>>1), "")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	expandMode, ok := parseExpandMode(*expand)
	if !ok {
		fmt.Fprintln(errOut, "invalid -expand-paths (expected off|safe)")
		return 2
	}
	foldMode, ok := parseFoldMode(*fold)
	if !ok {
		fmt.Fprintln(errOut, "invalid -fold (expected off|safe)")
		return 2
	}
	dd, ok := parseDelim(*docDelim)
	if !ok {
		fmt.Fprintln(errOut, "invalid -doc-delim (expected ,|tab|pipe)")
		return 2
	}
	ad, ok := parseDelim(*arrDelim)
	if !ok {
		fmt.Fprintln(errOut, "invalid -array-delim (expected ,|tab|pipe)")
		return 2
	}

	d, err := toon.NewDecoder(in,
		toon.WithStrict(*strict),
		toon.WithDecoderIndent(*indent),
		toon.WithExpandPaths(expandMode),
	)
	if err != nil {
		fmt.Fprintln(errOut, err.Error())
		return 1
	}

	v, err := d.Decode()
	if err != nil {
		fmt.Fprintln(errOut, err.Error())
		return 1
	}

	if cmd == "validate" {
		return 0
	}

	if cmd != "fmt" {
		fmt.Fprintln(errOut, "unknown command (expected fmt|validate)")
		return 2
	}

	e, err := toon.NewEncoder(out,
		toon.WithEncoderIndent(*indent),
		toon.WithDocDelimiter(dd),
		toon.WithArrayDelimiter(ad),
		toon.WithKeyFolding(foldMode),
		toon.WithFlattenDepth(*flattenDepth),
	)
	if err != nil {
		fmt.Fprintln(errOut, err.Error())
		return 1
	}
	if err := e.Encode(v); err != nil {
		fmt.Fprintln(errOut, err.Error())
		return 1
	}
	return 0
}

func parseDelim(s string) (toon.Delimiter, bool) {
	switch s {
	case ",":
		return toon.Comma, true
	case "tab":
		return toon.Tab, true
	case "pipe":
		return toon.Pipe, true
	default:
		return 0, false
	}
}

func parseExpandMode(s string) (toon.ExpandPathsMode, bool) {
	switch s {
	case "off":
		return toon.ExpandPathsOff, true
	case "safe":
		return toon.ExpandPathsSafe, true
	default:
		return "", false
	}
}

func parseFoldMode(s string) (toon.KeyFoldingMode, bool) {
	switch s {
	case "off":
		return toon.KeyFoldingOff, true
	case "safe":
		return toon.KeyFoldingSafe, true
	default:
		return "", false
	}
}
