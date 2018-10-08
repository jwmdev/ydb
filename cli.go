package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func cliParseStart(args []string) {
	startCommand := flag.NewFlagSet("start", flag.ExitOnError)
	tmp := startCommand.Bool("tmp", false, "Use a temporary directory for persisting data (content is lost when server stops)")
	dir := startCommand.String("dir", "", "Directory that is used to persist data")

	startCommand.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: ydb start [--dir dir] [--tmp]\n\n")
		startCommand.PrintDefaults()
	}
	startCommand.Parse(args)
	flag.Parse()
	if *tmp && *dir != "" {
		fmt.Fprintf(os.Stderr, "ydb: must not set --dir \"%s\" when using --tmp \n", *dir)
		os.Exit(1)
	}
	if *tmp && *dir == "" {
		tmpdir, err := ioutil.TempDir("", "ydb")
		if err != nil {
			fmt.Fprintln(os.Stderr, "ydb: unable to create temporary directory")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "using temporary directory")
		fmt.Fprintln(os.Stderr, "warning: data will be lost when server stops!")
		*dir = tmpdir
	}
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "ydb: missing --dir operand")
		fmt.Fprintln(os.Stderr, "Try 'ydb start --help' for more information")
		os.Exit(1)
	}
	if len(startCommand.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "ydb: too many arguments")
		fmt.Fprintln(os.Stderr, "Try 'ydb start --help' for more information")
		os.Exit(1)
	}
	initYdb(*dir)
	setupWebsocketsListener(":8899")
}

func main() {
	version := flag.Bool("version", false, "Print the cli version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [--version] <command> [<args>]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "available commands:\n")
		fmt.Fprintf(os.Stderr, "   start     Start a Ydb instance\n")
		fmt.Fprintf(os.Stderr, "   cli       Retrieve and modify content of a Ydb instance\n")
		fmt.Fprintf(os.Stderr, "   stats     Print live stats about a Ydb instance\n")
	}
	if *version {
		fmt.Println("ydb version 0.0.0") // TODO
		return
	}
	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "start":
		cliParseStart(os.Args[2:])
	default:
		flag.Usage()
		os.Exit(1)
	}
}
