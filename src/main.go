package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"./board"
)

const (
	version = "v2-Alpha"
)

func main() {

	Scorebot := flag.String("s", "", "Scorebot core address.")
	Directory := flag.String("d", "", "Scoreboard HTML directory path.")

	Listen := flag.String("l", "0.0.0.0:8080", "Address and port to listen on. (optional)")

	Tick := flag.Int("r", 5, "Scoreboard poll rate in seconds. (optional)")
	Timeout := flag.Int("t", 5, "Scoreboard request timeout in seconds. (optional)")

	LogFile := flag.String("o", "", "Scoreboard log file. (optional)")
	LogLevel := flag.Int("n", 2, "Scoreboard log level. (optional)")

	flag.Usage = func() {
		fmt.Printf("Scorebot Scoreboard %s\n\nUsage:\n", version)
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(*Directory) == 0 {
		if d, err := filepath.Abs(filepath.Dir(os.Args[0])); err == nil {
			*Directory = d
		}
	}
	if len(*Scorebot) == 0 || len(*Listen) == 0 || *Tick <= 0 || *Timeout < 0 {
		flag.Usage()
		os.Exit(1)
	}

	board, err := board.NewScoreboard(*Listen, *Timeout, *Tick, *Directory, *Scorebot, *LogFile, *LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "An error occured scoreboard creation: %s\n", err.Error())
		os.Exit(1)
	}

	if err := board.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "An error occured during operation: %s\n", err.Error())
		os.Exit(1)
	}
}
