// Copyright(C) 2020 iDigitalFlame
//
// This program is free software: you can redistribute it and / or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.If not, see <https://www.gnu.org/licenses/>.
//

package scoreboard

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/PurpleSec/logx"
)

const defaults = `{
    "log": {
        "file": "scoreboard.log",
        "level": 2
    },
    "tick": 5,
    "assets": "",
    "listen": "0.0.0.0:8080",
    "twitter": {
        "filter": {
            "language": [
                "en"
            ],
            "keywords": [
                "pvj",
                "ctf"
            ],
            "only_users": [],
            "blocked_users": [],
            "banned_words": []
        },
        "expire": 45,
        "auth": {
            "access_key": "",
            "consumer_key": "",
            "access_secret": "",
            "consumer_secret": ""
        }
    },
    "timeout": 10,
    "scorebot": "http://scorebot"
}
`
const usage = `Scorebot Scoreboard v2.22

Leaving any of the required Twitter options empty in command
line or config will result in Twitter functionality being disabled.
Required Twitter options: 'Consumer Key and Secret', 'Access Key and Secret',
'Twitter Keywords' and 'Twitter Language'.

Usage of scoreboard:
  -c <file>                 Scorebot configuration file path.
  -d                        Print default configuration and exit.
  -sbe <url>                Scorebot core address or URL (Required without "-c").
  -assets <dir>             Scoreboard secondary assets override URL.
  -dir <directory>          Scoreboard HTML override directory path.
  -log <file>               Scoreboard log file path.
  -log-level <number [0-5]> Scoreboard logging level (Default 2).
  -tick <seconds>           Scorebot poll tate, in seconds (Default 5).
  -timeout <seconds>        Scoreboard request timeout, in seconds (Default 10).
  -bind <socket>            Address and port to listen on (Default "0.0.0.0:8080").
  -cert <file>              Path to TLS certificate file.
  -key <file>               Path to TLS key file.
  -tw-ck <key>              Twitter Consumer API key.
  -tw-cs <secret>           Twitter Consumer API secret.
  -tw-ak <key>              Twitter Access API key.
  -tw-as <secret>           Twitter Access API secret.
  -tw-keywords <list>       Twitter search keywords (Comma separated)
  -tw-lang <list>           Twitter search language (Comma separated)
  -tw-expire <seconds>      Tweet display time, in seconds (Default 45).
  -tw-block-words <list>    Twitter blocked words (Comma separated).
  -tw-block-user <list>     Twitter blocked Usernames (Comma separated).
  -tw-only-users <list>     Twitter whitelisted Usernames (Comma separated).

Copyright (C) 2020 iDigitalFlame

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

`

type log struct {
	File  string `json:"file,omitempty"`
	Level int    `json:"level"`
}
type creds struct {
	AccessKey      string `json:"access_key"`
	ConsumerKey    string `json:"consumer_key"`
	AccessSecret   string `json:"access_secret"`
	ConsumerSecret string `json:"consumer_secret"`
}
type tweets struct {
	Credentials creds  `json:"auth"`
	Filter      filter `json:"filter"`
	Expire      int    `json:"expire"`
}
type config struct {
	Scorebot  string `json:"scorebot"`
	Key       string `json:"key,omitempty"`
	Cert      string `json:"cert,omitempty"`
	Directory string `json:"dir,omitempty"`
	Assets    string `json:"assets"`
	Listen    string `json:"listen"`
	Log       log    `json:"log,omitempty"`
	Twitter   tweets `json:"twitter,omitempty"`
	Timeout   int    `json:"timeout"`
	Tick      int    `json:"tick"`
	twitter   bool
}
type filter struct {
	Language     []string `json:"language"`
	Keywords     []string `json:"keywords"`
	OnlyUsers    []string `json:"only_users"`
	BlockedUsers []string `json:"blocked_users"`
	BlockedWords []string `json:"banned_words"`
}

func split(s string) []string {
	if len(s) == 0 {
		return []string{}
	}
	o := strings.Split(s, ",")
	for i := range o {
		o[i] = strings.TrimSpace(o[i])
	}
	return o
}
func (e errval) Error() string {
	if e.e == nil {
		return e.s
	}
	return e.s + ": " + e.e.Error()
}
func (e errval) Unwrap() error {
	return e.e
}
func (c *config) verify() error {
	if c.Tick <= 0 {
		return &errval{s: "tick " + strconv.Itoa(c.Tick) + " cannot be less than or equal to zero"}
	}
	if c.Timeout <= 0 {
		return &errval{s: "timeout " + strconv.Itoa(c.Timeout) + " cannot be less than or equal to zero"}
	}
	if c.Log.Level < int(logx.Trace) || c.Log.Level > int(logx.Fatal) {
		return &errval{s: "log level " + strconv.Itoa(c.Tick) + "  must be between zero and five"}
	}
	if len(c.Listen) == 0 {
		c.Listen = "0.0.0.0:8080"
	}
	if c.twitter = true; len(c.Twitter.Filter.Language) == 0 || len(c.Twitter.Filter.Keywords) == 0 {
		c.twitter = false
	}
	if len(c.Twitter.Credentials.AccessKey) == 0 || len(c.Twitter.Credentials.AccessSecret) == 0 {
		c.twitter = false
	}
	if len(c.Twitter.Credentials.ConsumerKey) == 0 || len(c.Twitter.Credentials.ConsumerSecret) == 0 {
		c.twitter = false
	}
	if c.twitter && c.Twitter.Expire <= 0 {
		return &errval{s: "tweet expire time " + strconv.Itoa(c.Timeout) + " cannot be less than or equal to zero"}
	}
	return nil
}

// Cmdline is a function that will create a Scoreboard instance from the supplied Cmdline
// parameters. This function will attempt to load the specified config file (if any) and fill in
// the proper settings. This function returns an error if any issues occur. If both returns are nil
// this means that the defaults are being printed and to bail out with a success status.
func Cmdline() (*Scoreboard, error) {
	var (
		c                     config
		d                     bool
		args                  = flag.NewFlagSet("Scorebot Scoreboard", flag.ExitOnError)
		twbWords, twoUsers    string
		s, twk, twl, twbUsers string
	)
	args.Usage = func() {
		os.Stdout.WriteString(usage)
		os.Exit(2)
	}
	args.StringVar(&s, "c", "", "scoreboard config file path.")
	args.BoolVar(&d, "d", false, "Print default configuration and exit.")
	args.StringVar(&c.Scorebot, "sbe", "", "Scorebot core address or URL (Required without -c).")
	args.StringVar(&c.Assets, "assets", "", "Scoreboard secondary assets override URL.")
	args.StringVar(&c.Directory, "dir", "", "Scoreboard HTML directory path.")
	args.StringVar(&c.Log.File, "log", "", "Scoreboard log file path.")
	args.IntVar(&c.Log.Level, "log-level", 2, "Scoreboard logging level (Default 2).")
	args.IntVar(&c.Tick, "tick", 5, "Scorebot poll rate, in seconds (Default 5).")
	args.IntVar(&c.Timeout, "timeout", 10, "Scoreboard request timeout, in seconds (Default 10).")
	args.StringVar(&c.Listen, "bind", "0.0.0.0:8080", "Address and port to listen on (Default 0.0.0.0:8080).")
	args.StringVar(&c.Key, "key", "", "Path to TLS key file.")
	args.StringVar(&c.Cert, "cert", "", "Path to TLS certificate file.")
	args.StringVar(&c.Twitter.Credentials.ConsumerKey, "tw-ck", "", "Twitter Consumer API key.")
	args.StringVar(&c.Twitter.Credentials.ConsumerSecret, "tw-cs", "", "Twitter Consumer API secret.")
	args.StringVar(&c.Twitter.Credentials.AccessKey, "tw-ak", "", "Twitter Access API key.")
	args.StringVar(&c.Twitter.Credentials.AccessSecret, "tw-as", "", "Twitter Access API secret.")
	args.StringVar(&twk, "tw-keywords", "", "Twitter search keywords (Comma separated)")
	args.StringVar(&twl, "tw-lang", "", "Twitter search language (Comma separated)")
	args.IntVar(&c.Twitter.Expire, "tw-expire", 45, "Tweet display time, in seconds (Default 45).")
	args.StringVar(&twbWords, "tw-block-words", "", "Twitter blocked words (Comma separated).")
	args.StringVar(&twbUsers, "tw-block-user", "", "Twitter blocked Usernames (Comma separated).")
	args.StringVar(&twoUsers, "tw-only-users", "", "Twitter whitelisted Usernames (Comma separated).")

	if err := args.Parse(os.Args[1:]); err != nil {
		os.Stdout.WriteString(usage)
		return nil, flag.ErrHelp
	}
	if d {
		os.Stdout.WriteString(defaults)
		return nil, nil
	}
	if len(s) == 0 && len(c.Scorebot) == 0 {
		os.Stdout.WriteString(usage)
		return nil, flag.ErrHelp
	}
	c.Twitter.Filter.OnlyUsers = split(twoUsers)
	c.Twitter.Filter.Language, c.Twitter.Filter.Keywords = split(twl), split(twk)
	c.Twitter.Filter.BlockedUsers, c.Twitter.Filter.BlockedWords = split(twbUsers), split(twbWords)
	if len(s) > 0 {
		b, err := ioutil.ReadFile(s)
		if err != nil {
			return nil, &errval{s: `cannot read file "` + s + `"`, e: err}
		}
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, &errval{s: `cannot parse JSON from file "` + s + `"`, e: err}
		}
	}
	return c.New()
}
