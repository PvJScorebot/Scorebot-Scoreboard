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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/PurpleSec/logx"
)

const (
	defaultTick     = 5
	defaultExpire   = 45
	defaultListen   = "0.0.0.0:8080"
	defaultTimeout  = 10
	defaultloglevel = int(logx.Warning)
)

var errInvalidNumber = errors.New("specified number is invalid")

type config struct {
	Log       configLog     `json:"log,omitempty"`
	Key       string        `json:"key,omitempty"`
	Cert      string        `json:"cert,omitempty"`
	Tick      int           `json:"tick"`
	Assets    string        `json:"assets"`
	Listen    string        `json:"listen"`
	Twitter   configTwitter `json:"twitter,omitempty"`
	Timeout   int           `json:"timeout"`
	Scorebot  string        `json:"scorebot"`
	Directory string        `json:"dir,omitempty"`

	twitter bool
}
type configLog struct {
	File  string `json:"file,omitempty"`
	Level int    `json:"level"`
}
type configCreds struct {
	AccessKey      string `json:"access_key"`
	ConsumerKey    string `json:"consumer_key"`
	AccessSecret   string `json:"access_secret"`
	ConsumerSecret string `json:"consumer_secret"`
}
type configFilter struct {
	Language     []string `json:"language"`
	Keywords     []string `json:"keywords"`
	OnlyUsers    []string `json:"only_users"`
	BlockedUsers []string `json:"blocked_users"`
	BlockedWords []string `json:"banned_words"`
}
type configTwitter struct {
	Filter      configFilter `json:"filter"`
	Expire      int          `json:"expire"`
	Credentials configCreds  `json:"auth"`
}

func defaults() {
	var c config
	c.Listen = defaultListen
	c.Scorebot = "http://scorebot"
	c.Twitter.Expire = defaultExpire
	c.Twitter.Filter.OnlyUsers = []string{}
	c.Twitter.Filter.Language = []string{"en"}
	c.Log.File, c.Log.Level = "scoreboard.log", 2
	c.Twitter.Filter.Keywords = []string{"pvj", "ctf"}
	c.Tick, c.Timeout, c.Twitter.Expire = defaultTick, defaultTimeout, defaultExpire
	c.Twitter.Filter.BlockedUsers, c.Twitter.Filter.BlockedWords = []string{}, []string{}
	b, _ := json.MarshalIndent(c, "", "    ")
	fmt.Fprintln(os.Stdout, string(b))
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
func (c *config) verify() error {
	if c.Tick <= 0 {
		return fmt.Errorf("tick cannot be less than or equal to zero: %w", errInvalidNumber)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout cannot be less than or equal to zero: %w", errInvalidNumber)
	}
	if c.Log.Level < int(logx.Trace) || c.Log.Level > int(logx.Fatal) {
		return fmt.Errorf("log level must be between zero and five: %w", errInvalidNumber)
	}
	if len(c.Listen) == 0 {
		c.Listen = defaultListen
	}
	c.twitter = true
	if len(c.Twitter.Filter.Language) == 0 || len(c.Twitter.Filter.Keywords) == 0 {
		c.twitter = false
	}
	if len(c.Twitter.Credentials.AccessKey) == 0 || len(c.Twitter.Credentials.AccessSecret) == 0 {
		c.twitter = false
	}
	if len(c.Twitter.Credentials.ConsumerKey) == 0 || len(c.Twitter.Credentials.ConsumerSecret) == 0 {
		c.twitter = false
	}
	if c.twitter && c.Twitter.Expire <= 0 {
		return fmt.Errorf("tweet expire time cannot be less than or equal to zero: %w", errInvalidNumber)
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
		args                  = flag.NewFlagSet(fmt.Sprintf("Scorebot Scoreboard v%.1f", version), flag.ExitOnError)
		twbWords, twoUsers    string
		s, twk, twl, twbUsers string
	)
	args.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, version, os.Args[0])
		os.Exit(2)
	}
	args.StringVar(&s, "c", "", "scoreboard config file path.")
	args.BoolVar(&d, "d", false, "Print default configuration and exit.")
	args.StringVar(&c.Scorebot, "sbe", "", "Scorebot core address or URL (Required without -c).")
	args.StringVar(&c.Assets, "assets", "", "Scoreboard secondary assets override URL.")
	args.StringVar(&c.Directory, "dir", "", "Scoreboard HTML directory path.")
	args.StringVar(&c.Log.File, "log", "", "Scoreboard log file path.")
	args.IntVar(&c.Log.Level, "log-level", defaultloglevel, "Scoreboard logging level (Default 2).")
	args.IntVar(&c.Tick, "tick", defaultTick, "Scorebot poll rate, in seconds (Default 5).")
	args.IntVar(&c.Timeout, "timeout", defaultTimeout, "Scoreboard request timeout, in seconds (Default 10).")
	args.StringVar(&c.Listen, "bind", defaultListen, "Address and port to listen on (Default 0.0.0.0:8080).")
	args.StringVar(&c.Key, "key", "", "Path to TLS key file.")
	args.StringVar(&c.Cert, "cert", "", "Path to TLS certificate file.")
	args.StringVar(&c.Twitter.Credentials.ConsumerKey, "tw-ck", "", "Twitter Consumer API key.")
	args.StringVar(&c.Twitter.Credentials.ConsumerSecret, "tw-cs", "", "Twitter Consumer API secret.")
	args.StringVar(&c.Twitter.Credentials.AccessKey, "tw-ak", "", "Twitter Access API key.")
	args.StringVar(&c.Twitter.Credentials.AccessSecret, "tw-as", "", "Twitter Access API secret.")
	args.StringVar(&twk, "tw-keywords", "", "Twitter search keywords (Comma separated)")
	args.StringVar(&twl, "tw-lang", "", "Twitter search language (Comma separated)")
	args.IntVar(&c.Twitter.Expire, "tw-expire", defaultExpire, "Tweet display time, in seconds (Default 45).")
	args.StringVar(&twbWords, "tw-block-words", "", "Twitter blocked words (Comma separated).")
	args.StringVar(&twbUsers, "tw-block-user", "", "Twitter blocked Usernames (Comma separated).")
	args.StringVar(&twoUsers, "tw-only-users", "", "Twitter whitelisted Usernames (Comma separated).")
	if err := args.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, usage, version, os.Args[0])
		return nil, flag.ErrHelp
	}
	if d {
		defaults()
		return nil, nil
	}
	if len(s) == 0 && len(c.Scorebot) == 0 {
		fmt.Fprintf(os.Stderr, usage, version, os.Args[0])
		return nil, flag.ErrHelp
	}
	c.Twitter.Filter.OnlyUsers = split(twoUsers)
	c.Twitter.Filter.Language, c.Twitter.Filter.Keywords = split(twl), split(twk)
	c.Twitter.Filter.BlockedUsers, c.Twitter.Filter.BlockedWords = split(twbUsers), split(twbWords)
	if len(s) > 0 {
		if err := loadFile(s, &c); err != nil {
			return nil, err
		}
	}
	if err := c.verify(); err != nil {
		return nil, err
	}
	return c.new()
}
func loadFile(s string, c *config) error {
	var (
		p      = os.ExpandEnv(s)
		f, err = os.Stat(p)
	)
	if err != nil {
		return fmt.Errorf("cannot load file %q: %w", p, err)
	}
	if f.IsDir() {
		return fmt.Errorf("cannot load %q: path is not a file", p)
	}
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return fmt.Errorf("cannot read file %q: %w", p, err)
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("cannot read file %q from JSON: %w", p, err)
	}
	return nil
}
