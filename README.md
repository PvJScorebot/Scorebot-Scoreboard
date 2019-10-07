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
cd scoreboard
go build ./..
cd ..
bash build.sh
```

## Parameters

The Scorebord can be configured by command line options, though it's preferred to use a config file instead. (Below).

```text
Scorebot Scoreboard v1.7

Usage:
  -assets string
        Secondary Assets Override URL.
  -bind string
        Address and Port to Listen on. (default "0.0.0.0:8080")
  -c string
        Scorebot Config File Path.
  -cert string
        Path to TLS Certificate File.
  -d    Print Default Config and Exit.
  -dir string
        Scoreboard HTML Directory Path.
  -key string
        Path to TLS Key File.
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
        Twitter Blocked Usernames. (comma separated)
  -tw-block-words string
        Twitter Blocked Words. (comma separated)
  -tw-ck string
        Twitter Consumer API Key.
  -tw-cs string
        Twitter Consumer API Secret.
  -tw-expire int
        Tweet Display Time. (in seconds) (default 45)
  -tw-keywords string
        Twitter Search Keywords. (comma separated)
  -tw-lang string
        Twitter Search Language. (comma separated)
  -tw-only-users string
        Twitter WHitelisted Usernames. (comma separated)
```

## Config File

The best way to configure the scoreboard is to use a config file. This file will override any command line options.
To run with the config file use the command line option: `-c <file_path>`

Default Config:

```json
{
    "log": {
        "file": "",
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
        "timeout": 10,
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
