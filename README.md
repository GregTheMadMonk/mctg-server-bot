# server-bot

A simple bot that provides basic integration between a Minecraft server and a
Telegram channel

Written in Go as a learning exercise

## Usage

The bot executes the Minecraft server as a child subprocess from the provided
command-line and attaches to the server's `stdin`/`stdout`

The program does not accept any arguments.
All configuration is loaded from the `mctg-bot-config.json` file in the
program's working directory.
The config's format is as follows:

```json
{
    "bot": {
        "api_token":      "the bot's Telegram API token",
        "chat_id":        the_channel_for_your_bot_to_live_in,
        "admin_username": "(Telegram) name of the user that can issue slash-commands. The server will start without it, but you will not be able to gracefully kill it with /kill-server"
    },
    "server": {
        "cmdline":   [ "bash", "run.sh", "nogui" ],
        "log_lines": unused_for_now_number_of_log_lines_the_server_stores_in_ram
    }
}
```

On startup, the bot also starts the Minecraft server in a child process using
`server.cmdline` from the config.

If a Telegram user's message starts with a slash (`/`), the server performs
a check.
If the message is one of the user-allowed slash commands, the action is
performed:

|Command|Action|
|---|---|
|`/players`|List players online on the server|

If the user is an admin (as specified in the config), the following commands
are also available to them:
|Command|Action|
|---|---|
|`/kill-server`|Send `/stop` to the server console and don't restart when it exits. Also kills the bot when the server terminates|
|Any other message starting with `/`|Passed directly to the server's `stdin`|

In every other scenareo, the message is interpreted as a simple message and
is sent to the server as if the user was just talking with a `/say` command.

## Mod

The `mod/` subdirectory contains an accompanying server mod.
It enhances chat messages (makes both in-game and from-telegram chat messages
uniformely colored) and allows the bot to send player death messages to the
channel (ofc it was technically possible by just observing logs - but there
is no way I'd be caught hardcoding all possible death messages into a regex)
