package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"./board"
	"./board/web"
)

const (
	version = "v2-Alpha"
)

func main() {
	ConfigFile := flag.String("c", "", "Scorebot Config File Path.")
	ConfigDefault := flag.Bool("d", false, "Print Default Config and Exit.")

	LogFile := flag.String("log", "", "Scoreboard Log File Path.")
	LogLevel := flag.Int("log-level", int(board.DefaultLogLevel), "Scoreboard Log Level.")

	Tick := flag.Int("tick", int(board.DefaultTick), "Scoreboard Poll Rate. (in seconds)")

	Listen := flag.String("bind", board.DefaultListen, "Address and Port to Listen on.")

	Timeout := flag.Int("timeout", int(board.DefaultTimeout), "Scoreboard Request Timeout. (in seconds)")

	Scorebot := flag.String("sbe", "", "Scorebot Core Address or URL.")

	Directory := flag.String("dir", "", "Scoreboard HTML Directory Path.")

	TwitterCosumerKey := flag.String("tw-ck", "", "Twitter Consumer API Key.")
	TwitterCosumerSecret := flag.String("tw-cs", "", "Twitter Consumer API Secret.")

	TwitterAccessKey := flag.String("tw-ak", "", "Twitter Access API Key.")
	TwitterAccessSecret := flag.String("tw-as", "", "Twitter Access API Secret.")

	TwitterKeywords := flag.String("tw-keywords", "", "Twitter Search Keywords. (comma seperated)")
	TwitterLanguage := flag.String("tw-lang", "", "Twitter Search Lanugage. (comma seperated)")

	TwitterExpire := flag.Int("tw-expire", int(board.DefaultExpire), "Tweet Display Time. (in seconds)")

	TwitterBlockWords := flag.String("tw-block-words", "", "Twitter Blocked Words. (comma seperated)")
	TwitterBlockUsers := flag.String("tw-block-user", "", "Twitter Blocked Usernames. (comma seperated)")

	TwitterOnlyUsers := flag.String("tw-only-users", "", "Twitter WHitelisted Usernames. (comma seperated)")

	flag.Usage = func() {
		fmt.Printf("Scorebot Scoreboard %s\n\nUsage:\n", version)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *ConfigDefault {
		fmt.Printf("%s\n", board.Defaults())
		os.Exit(0)
	}

	var c *board.Config

	if len(*ConfigFile) > 0 {
		var err error
		c, err = board.Load(*ConfigFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
	} else {
		if len(*Directory) == 0 {
			if d, err := filepath.Abs(filepath.Dir(os.Args[0])); err == nil {
				*Directory = d
			}
		}
		if len(*Scorebot) == 0 || len(*Listen) == 0 || *Tick <= 0 || *Timeout < 0 || *TwitterExpire <= 0 {
			flag.Usage()
			os.Exit(2)
		}
		c = &board.Config{
			Log: &board.Log{
				File:  *LogFile,
				Level: uint8(*LogLevel),
			},
			Tick:   uint16(*Tick),
			Listen: *Listen,
			Twitter: &board.Twitter{
				Filter: &web.Filter{
					Language:     board.SplitParm(*TwitterLanguage, board.ConfigSeperator),
					Keywords:     board.SplitParm(*TwitterKeywords, board.ConfigSeperator),
					OnlyUsers:    board.SplitParm(*TwitterOnlyUsers, board.ConfigSeperator),
					BlockedUsers: board.SplitParm(*TwitterBlockUsers, board.ConfigSeperator),
					BlockedWords: board.SplitParm(*TwitterBlockWords, board.ConfigSeperator),
				},
				Expire:  uint16(*TwitterExpire),
				Timeout: uint16(*Timeout),
				Credentials: &web.Credentials{
					AccessKey:      *TwitterAccessKey,
					ConsumerKey:    *TwitterCosumerKey,
					AccessSecret:   *TwitterAccessSecret,
					ConsumerSecret: *TwitterCosumerSecret,
				},
			},
			Timeout:   uint16(*Timeout),
			Scorebot:  *Scorebot,
			Directory: *Directory,
		}
	}

	board, err := board.NewScoreboard(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	if err := board.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
