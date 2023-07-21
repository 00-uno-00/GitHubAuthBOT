# GitHubAuthBot

A simple telegram bot to automate the process of adding collaborators to different repositories, created with the objective of distributing university notes only between the intended users.

## Features

- 2FA through email
- Easy to configure permissions and repositries  

## Setup and Deployment

For security reasons I have decided to pass sensible data throughtout enviornment variables. 

Before building and executing the code it's necessary to set up said enviornment variables, the bot's API token will be provided by [@BotFather](https://t.me/BotFather) upon it's creation. While you can use [this short guide](https://github.blog/2013-05-16-personal-api-tokens/) to get your personal access token (you'll need to select full repository control).

Having obtained both tokens you can set up the envoirnment variables with the following commands in your terminal. 

```bash
env export GAB_TG_GITHUB_API         = <your telegram HTTP API access token>

env export GAB_GITHUB_ACCESS_TOKEN   = <your GitHub HTTP API access token>
```
For the 2FA you'll need to set up gomail  with your preferred email client 
```bash
env export GAB_EMAIL_PASSW           = <your email password>

env export GAB_EMAIL_USERNAME        = <your email username>

env export GAB_SMTP_PORT             = <your smtp port>

env export GAB_SMTP_HOST             = <your smtp host>
```
> Note that you'll need to take some [extra setps](https://github.com/go-gomail/gomail/issues/28) to set up a google account with 2FA active as sender.  

(you can use `env` to check if they have been set up correctly)

```bash
go build
./GitHubAuthBOT
```
should get the bot started.
## TODO

- [x] Add function and relative bot command to list all of the owner's repositories the user has access to

- [x] Add 2FA to verify the provided email

- [ ] add addblacklist

- [ ] add delblacklist

- [ ] add addrepository

- [ ] add delrepository

- [ ] add securitylvl

- [ ] add user