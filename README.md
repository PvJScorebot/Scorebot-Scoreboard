# Scorebot Scoreboard Service

Scorebot Client for [Scorebot-Core](https://github.com/iDigitalFlame/scorebot-core)

This client is new and improved and allows for multiple games to be synced and displayed
using synced websockets.

This scoreboard supports the Scorebot > 3.3.4 events API and is capable of displaying videos, messages and event
window popups.

Twitter integration is also enabled in this version. Add your API keys into the config to enable it.

## Download

You can visit the Releases page to download a linux compiled version. The resources and images are packed into the
binary using [Packr](https://github.com/gobuffalo/packr/tree/master/v2). It will run as long as you point it to an active Scorebot instance.

## Building

```shell
git clone https://github.com/iDigitalFlame/scoreboard
cd scoreboard
go build ./..
bash build.sh
```

## Parameters

The Scorebord can be configured by command line options, though it's preferred to use a config file instead. (Below).

```text
Scorebot Scoreboard v1.2

Usage:
  -bind string
        Address and Port to Listen on. (default "0.0.0.0:8080")
  -c sting
        Scorebot Config File Path.
  -d    Print Default Config and Exit.
  -dir string
        Scoreboard HTML Directory Path.
  -log string
        Scoreboard Log File Path.
  -log-level int
        Scoreboard Log Level. (default 2)
  -sbe string
        Scorebot Core Address or URL.
  -tick int
        Scoreboard Poll Rate. (in seconds) (default 5)
  -timeout int
        Scoreboard Request Timeout. (in seconds) (default 10)
  -tw-ak string
        Twitter Access API Key.
  -tw-as string
        Twitter Access API Secret.
  -tw-block-user string
        Twitter Blocked Usernames. (comma seperated)
  -tw-block-words string
        Twitter Blocked Words. (comma seperated)
  -tw-ck string
        Twitter Consumer API Key.
  -tw-cs string
        Twitter Consumer API Secret.
  -tw-expire int
        Tweet Display Time. (in seconds) (default 45)
  -tw-keywords string
        Twitter Search Keywords. (comma seperated)
  -tw-lang string
        Twitter Search Lanugage. (comma seperated)
  -tw-only-users string
        Twitter WHitelisted Usernames. (comma seperated)
```

## Config File

The best way to confiure the scoreboard is to use a config file. This file will override any command line options.
To run with the config file use the command line option: `-c <filepath>`

Default Config:

```json
{
    "log": {
        "file": "",
        "level": 2
    },
    "tick": 5,
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
        "timeout": 10,
        "auth": {
            "access_key": "",
            "consomer_key": "",
            "access_secret": "",
            "consomer_secret": ""
        }
    },
    "timeout": 10,
    "scorebot": "http://scorebot",
    "dir": "html"
}
```
