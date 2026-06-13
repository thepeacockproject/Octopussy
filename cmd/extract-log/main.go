package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: extract-log <logfile>")
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		return fmt.Errorf("missing log file argument")
	}

	logFile := flag.Arg(0)

	if _, err := os.Stat(logFile); err != nil {
		return fmt.Errorf("log file not found: %s", logFile)
	}

	name := strings.TrimSuffix(filepath.Base(logFile), filepath.Ext(logFile))
	outputDir := filepath.Join(filepath.Dir(logFile), name)

	lines, err := readLines(logFile)
	if err != nil {
		return err
	}

	exchanges := parseExchanges(lines)

	written, err := writeExchanges(exchanges, outputDir)
	if err != nil {
		return err
	}

	fmt.Printf("Parsed %d request/response pair(s) from %s\n", len(exchanges), filepath.Base(logFile))
	fmt.Printf("Wrote %d JSON file(s) to %s\n", written, outputDir)

	return nil
}
