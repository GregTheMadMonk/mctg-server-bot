# server-bot

A simple bot that provides basic integration between a Minecraft server and a
Telegram channel

Written in Go as a learning exercise. Very basic.

## Usage

The bot executes the Minecraft server as a child subprocess from the provided
command-line and attaches to the server's `stdin`/`stdout`

```bash
go run main.go [server-command-line]
```

e.g. in case of Forge:

```bash
go run main.go bash run.sh nogui
```

The bot will not run if the following environment variables aren't set:
* `SERVERBOT_TOKEN` - Telegram API token for the bot
* `SERVERBOT_CHAT` - chat ID for the bot to live in
* `SERVERBOT_ADMIN` - (optional) username of the user that is able to give 
  slash-commands to the server (essentially rcon)

The only slash-command available to non-admins is `/players`. It prints the
list of players currently on the server
