# Scorebot Scoreboard Service

Scorebot Client for [Scorebot-Core](https://github.com/iDigitalFlame/scorebot-core)

This client is new and improved and allows for multiple games to be synced and displayed
using synced websockets.

This scoreboard supports the Scorebot > 3.3.4 events API and is capable of displaying videos, messages and event
window popups.

Twitter integration is also enabled in this version. Add your API keys into the config to enable it.

## Download

You can visit the [Releases](https://github.com/iDigitalFlame/scorebot-scoreboard/releases) page to download a linux compiled version. The resources and images are packed into the
binary using [Packr](https://github.com/gobuffalo/packr/tree/master/v2). It will run as long as you point it to an active Scorebot instance.

## Building

```shell
git clone https://github.com/iDigitalFlame/scorebot-scoreboard/
cd scorebot-scoreboard/scoreboard
go build ./..
cd ..
bash build.sh
```

## Parameters

The Scoreboard can be configured by command line options, though it's preferred to use a config file instead. (Below).

```text
Scorebot Scoreboard v2.2

Leaving any of the required Twitter options empty in command
line or config will result in Twitter functionality being disabled.
Required Twitter options: 'Consumer Key and Secret', 'Access Key and Secret',
'Twitter Keywords' and 'Twitter Language'.

Usage of scorebot-scoreboard:
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
```

## Config File

The best way to configure the scoreboard is to use a config file. This file will override any command line options.
To run with the config file use the command line option: `-c <file_path>`

Default Config:

```json
{
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
    "key": "",
    "scorebot": "http://scorebot",
    "cert": "",
    "dir": "html"
}
```
