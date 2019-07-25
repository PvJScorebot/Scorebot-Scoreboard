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
