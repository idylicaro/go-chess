// Package cli is the command-line client: the subcommands of cmd/cli.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// env is what a command reads and writes, so tests can drive it.
type env struct {
	ctx context.Context
	in  io.Reader
	out io.Writer
	err io.Writer
	now func() time.Time
	// getenv reads environment variables, for terminal detection.
	getenv func(string) string
}

// usageError is a mistake in how the command was invoked (exit status 2).
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error { return usageError{fmt.Sprintf(format, args...)} }

type command struct {
	name    string
	summary string
	run     func(e env, args []string) error
}

// commands lists the subcommands in help order.
var commands = []command{
	{"play", "play a game in the terminal", runPlay},
	{"analyze", "ask a UCI engine to analyze a position", runAnalyze},
	{"perft", "count the leaf nodes of the move tree", runPerft},
	{"fen", "check a FEN and show its position", runFEN},
	{"pgn", "read a PGN file and show its games", runPGN},
}

// Run executes the command line (without the program name) and returns the
// process exit status: 0 on success, 1 on failure and 2 on a usage error.
// With no arguments, or when the first one is a flag, it runs "play".
//
//	os.Exit(cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	return execute(env{ctx: ctx, in: in, out: out, err: errOut, now: time.Now, getenv: os.Getenv}, args)
}

// execute is Run with the environment spelled out, so tests can control it.
func execute(e env, args []string) int {
	out, errOut := e.out, e.err
	name := "play"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, rest = args[0], args[1:]
	} else if len(args) > 0 && (args[0] == "-h" || args[0] == "--help" || args[0] == "-help") {
		printUsage(out)
		return 0
	}
	if name == "help" {
		if len(rest) == 0 {
			printUsage(out)
			return 0
		}
		name, rest = rest[0], []string{"--help"}
	}
	for _, c := range commands {
		if c.name != name {
			continue
		}
		err := c.run(e, rest)
		switch {
		case err == nil, errors.Is(err, flag.ErrHelp):
			return 0
		case errors.As(err, new(usageError)):
			fmt.Fprintf(errOut, "chess %s: %v\nRun \"chess %s --help\" for usage.\n", c.name, err, c.name)
			return 2
		case errors.Is(err, context.Canceled):
			return 130
		default:
			fmt.Fprintf(errOut, "chess %s: %v\n", c.name, err)
			return 1
		}
	}
	fmt.Fprintf(errOut, "chess: unknown command %q\n\n", name)
	printUsage(errOut)
	return 2
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: chess <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	width := 0
	for _, c := range commands {
		width = max(width, len(c.name))
	}
	for _, c := range commands {
		fmt.Fprintf(w, "  %-*s  %s\n", width, c.name, c.summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "With no command, \"play\" starts. Run \"chess <command> --help\" for the flags of a command.")
}

// newFlagSet makes a flag set that prints usage to stderr and does not exit.
// synopsis is the part after "chess <name>", e.g. "[flags] <depth>".
func newFlagSet(e env, name, synopsis, description string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(e.err)
	fs.Usage = func() {
		fmt.Fprintf(e.err, "Usage: chess %s %s\n\n%s\n", name, synopsis, description)
		if hasFlags(fs) {
			fmt.Fprintln(e.err, "\nFlags:")
			fs.PrintDefaults()
		}
	}
	return fs
}

func hasFlags(fs *flag.FlagSet) bool {
	found := false
	fs.VisitAll(func(*flag.Flag) { found = true })
	return found
}

// parseFlags parses args; a flag error becomes a usageError, and -h/--help
// is flag.ErrHelp after printing the usage. It accepts flags after
// positional arguments, as in "perft 4 --divide".
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageError{"invalid flags"}
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

// readInput reads the named file, or standard input for "-".
func readInput(e env, path string) (string, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(e.in)
	} else {
		data, err = os.ReadFile(path)
	}
	return string(data), err
}
