# GitHubAuthBot

A simple telegram bot to automate the process of adding collaborators to different repositories, created with the objective of distributing university notes /without distributing material to outsiders therefore breaching copyright/only between the intended users.

## Setup and Deployment

For security reasons I have decided to pass sensible data throughtout enviornment variables. 

Before building and executing the code it's necessary to set up said enviornment variables, the bot's API token will be provided by [@BotFather](https://t.me/BotFather) upon it's creation. While you can use [this short guide](https://github.blog/2013-05-16-personal-api-tokens/) to get your personal access token (you'll need to select full repository control).

Having obtained both tokens you can set up the envoirnment variables with the following commands in your terminal. 

```bash
env export GAB_TG_GITHUB_API=<your telegram HTTP API access token>
```

```bash
env export GAB_GITHUB_ACCESS_TOKEN=<your GitHub HTTP API access token>
```
(you can use `env` to check if they have been set up correctly)

```bash
go build
./GitHubAuthBOT
```
should get the bot started.
## TODO

- Add function and relative bot command to list all of the owner's repositories the user has access to
