package main

import (
	"bufio"
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/gomail.v2"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

var (
	bot *tgbotapi.BotAPI

	owner      = "00-uno-00"
	Permission = "pull"

	TG_GITHUB_API       = os.Getenv("GAB_TG_GITHUB_API")
	GITHUB_ACCESS_TOKEN = os.Getenv("GAB_GITHUB_ACCESS_TOKEN")

	SMPT_HOST      = os.Getenv("GAB_SMTP_HOST")
	SMTP_PORT, _   = strconv.Atoi(os.Getenv("GAB_SMTP_PORT"))
	EMAIL_USERNAME = os.Getenv("GAB_EMAIL_USERNAME")
	EMAIL_PASSW    = os.Getenv("GAB_EMAIL_PASSW")
	ch             = make(chan *gomail.Message)

	REPOS = []string{"lft_lab", "archelab", "prog1", "prog2"}

	CODES = []int{}
)

type state struct {
	address    string
	ghusername string
	verified   bool
}

type syncMap struct {
	data map[int64]state
	m    sync.Mutex
}

func (s *syncMap) get(id int64) (state, error) {
	s.m.Lock()
	defer s.m.Unlock()

	data, ok := s.data[id]
	if !ok {
		return state{}, errors.New("state not found")
	}
	log.Println("got data")
	return data, nil
}

func (s *syncMap) set(id int64, ns state) {
	s.m.Lock()
	defer s.m.Unlock()
	s.data[id] = ns
	log.Println("set data")

}

func main() {
	var err error

	bot, err = tgbotapi.NewBotAPI(TG_GITHUB_API)
	if err != nil {
		// Abort if something is wrong
		log.Panic(err)
	}

	// Set up new email dialer
	dialer := gomail.NewDialer(SMPT_HOST, SMTP_PORT, EMAIL_USERNAME, EMAIL_PASSW)

	go emailSender(dialer)

	s := syncMap{map[int64]state{}, sync.Mutex{}}

	// Set this to true to log all interactions with telegram servers
	bot.Debug = false

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Create a new cancellable background context. Calling `cancel()` leads to the cancellation of the context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// `updates` is a golang channel which receives telegram updates
	updates := bot.GetUpdatesChan(u)

	//initaial message

	// Pass cancellable context to goroutine
	go receiveUpdates(ctx, updates, &s) //newcontext for each routine?

	// Tell the user the bot is online
	log.Println("Start listening for updates. Press enter to stop")

	// Wait for a newline symbol, then cancel handling updates
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()

}

func receiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, s *syncMap) {
	for {
		select {
		// stop looping if ctx is cancelled
		case <-ctx.Done():
			updates.Clear()
			return
		// receive update from channel and then handle it
		case update := <-updates:
			go handleMessage(ctx, update.Message, s)
		}
	}
}

func handleMessage(ctx context.Context, message *tgbotapi.Message, s *syncMap) {

	user := message.From
	text := message.Text

	if user == nil {
		return
	}

	// Print to console
	log.Println("handle message", text)
	var err error

	if message.IsCommand() {
		command := message.Command()
		args := strings.Fields(message.CommandArguments())
		log.Println("command:", command)
		hCommand(user.ID, command, args, s)
	} else if code, err := strconv.Atoi(message.Text); err == nil {
		for _, v := range CODES {
			if code == v {
				verifieduser, err := s.get(user.ID)
				if err != nil {
					log.Println("unexpected error")
					return
				}
				verifiedState := state{verifieduser.address, verifieduser.ghusername, true}
				s.set(user.ID, verifiedState)
			}
		}
		return
	} else {
		msg := tgbotapi.NewMessage(user.ID, "invalid message")
		bot.Send(msg)
	}

	if err != nil {
		log.Printf("An error occured: %s", err.Error())
	}
}

func hCommand(chatId int64, command string, arguments []string, s *syncMap) error {
	switch command {
	case "menu": // etc
		msg := tgbotapi.NewMessage(chatId, "I comandi disponibili sono: \n /verifica <username> <example@email.xyz> - verifica se puoi accedere alle repository \n /accedi <nome repo> - seleziona la repository a cui vuoi avere accesso \n /accessi - elenco degli accessi alle repository \n /gsora - colui che fornisce la VPS")
		bot.Send(msg)
		log.Println("menu")
	case "gsora":
		msg := tgbotapi.NewMessage(chatId, "gsora amico delle guardie (More info at: https://noskills.club)")
		bot.Send(msg)
	case "verifica":
		hVerifica(arguments, chatId, s)
	case "accessi":
		hAccessi(chatId, s)
	case "accedi":
		handleAccedi(chatId, arguments, s)
	case "aggiorna":
		handleAggiorna(arguments, chatId, s)
	case "start":
		msg := tgbotapi.NewMessage(chatId, "Scrivi /menu per vedere i comandi disponibili")
		bot.Send(msg)
	default:
		msg := tgbotapi.NewMessage(chatId, "Comando non riconosciuto")
		bot.Send(msg)
	}

	return nil
}

func hVerifica(args []string, chatID int64, s *syncMap) {
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(chatID, "inserisci comando con username GitHub e Mail (/verifica <username> <example@email.xyz>)")
		bot.Send(msg)
		return
	}

	user, err := s.get(chatID)
	if err != nil {
		if strings.HasSuffix(args[1], "@unito.it") || strings.HasSuffix(args[1], "@edu.unito.it") {
			go auth(chatID, args, s)
			return
		} else {
			msg := tgbotapi.NewMessage(chatID, "email invalida")
			bot.Send(msg)
			return
		}
	} else {
		msg := tgbotapi.NewMessage(chatID, "Utente gia' verificato, per aggiornare usa /aggiorna <username> <example@email.xyz> ")
		bot.Send(msg)
		msg = tgbotapi.NewMessage(chatID, "email: "+user.address)
		bot.Send(msg)
		msg = tgbotapi.NewMessage(chatID, "user: "+user.ghusername)
		bot.Send(msg)
	}
}

func handleAggiorna(args []string, chatID int64, s *syncMap) {
	if len(args) < 2 {
		msg := tgbotapi.NewMessage(chatID, "Inserisci comando con username GitHub e Mail (/aggiorna <username> <example@email.xyz>)")
		bot.Send(msg)
		return
	}
	user, _ := s.get(chatID)

	if strings.HasSuffix(args[1], "@unito.it") && user.verified || strings.HasSuffix(args[1], "@edu.unito.it") && user.verified {
		state := state{args[1], args[0], true}
		s.set(chatID, state)
		user, _ = s.get(chatID)
		msg := tgbotapi.NewMessage(chatID, "Dati aggiornati: "+user.ghusername)
		bot.Send(msg)
		return
	} else {
		msg := tgbotapi.NewMessage(chatID, "Email invalida")
		bot.Send(msg)
		return
	}
}

func handleAccedi(chatID int64, args []string, s *syncMap) {
	user, err := s.get(chatID)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "verifica non effettuata")
		bot.Send(msg)
		return
	} else if len(args) == 0 {
		rlist := strings.Join(REPOS, " \n -")
		msg := tgbotapi.NewMessage(chatID, "Sintassi comando: /accedi <RepoName>  Elenco delle repository: \n -"+rlist)
		bot.Send(msg)
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: GITHUB_ACCESS_TOKEN},
	)

	tc := oauth2.NewClient(context.Background(), ts)

	client := github.NewClient(tc)
	for i := 0; i < len(REPOS); i++ {
		if strings.EqualFold(args[0], REPOS[i]) {
			if !checkCollaborator(*client, user.ghusername, REPOS[i]) {
				log.Println("Adding collaborator...")
				if addCollaborator(*client, user.ghusername, REPOS[i]) != "201 Created" {
					msg := tgbotapi.NewMessage(chatID, addCollaborator(*client, user.ghusername, REPOS[i]))
					bot.Send(msg)
					return
				}
				msg := tgbotapi.NewMessage(chatID, "Invito inviato")
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatID, "Accesso gia' ottenuto")
				bot.Send(msg)
			}
			return
		}
		if i == len(REPOS)+1 {
			msg := tgbotapi.NewMessage(chatID, "Repo Invalida")
			bot.Send(msg)
		}
	}
}

func hAccessi(chatID int64, s *syncMap) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: GITHUB_ACCESS_TOKEN},
	)

	tc := oauth2.NewClient(context.Background(), ts)

	client := github.NewClient(tc)

	user, err := s.get(chatID)
	if err == nil {
		collaborates := []string{}
		for _, r := range REPOS {
			if checkCollaborator(*client, user.ghusername, r) {
				collaborates = append(collaborates, r)
			}
		}
		repolist := strings.Join(collaborates, ", ")
		msg := tgbotapi.NewMessage(chatID, "Repositories a cui hai accesso: "+repolist)
		bot.Send(msg)
	} else if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Utente non verificato")
		bot.Send(msg)
	}

}

func checkCollaborator(client github.Client, ghusername, RepoName string) bool {
	ret, resp, err := client.Repositories.IsCollaborator(context.Background(), owner, RepoName, ghusername)
	if err != nil {
		log.Println("errore: " + resp.Response.Status + "   " + err.Error())
	}
	return ret
}

func addCollaborator(client github.Client, ghusername, RepoName string) string {
	out, resp, err := client.Repositories.AddCollaborator(context.Background(), owner, RepoName, ghusername, &github.RepositoryAddCollaboratorOptions{Permission: "pull"})
	if err != nil {
		log.Println("errore: " + resp.Response.Status + "   " + err.Error())
		return resp.Response.Status
	}
	log.Printf("url:" + *out.URL)
	return resp.Response.Status
}

func emailSender(dialer *gomail.Dialer) {
	var s gomail.SendCloser
	var err error
	open := false

	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return
			}
			if !open {
				if s, err = dialer.Dial(); err != nil {
					panic(err)
				}
				open = true
			}
			if err := gomail.Send(s, m); err != nil {
				log.Print(err)
			}
		case <-time.After(15 * time.Second):
			if open {
				if err := s.Close(); err != nil {
					panic(err)
				}
				open = false
			}
		}
	}
}

func codeGenerator() int {
	code := rand.Intn(999999)
	if code%2 == 0 {
		return int(math.Sqrt(float64(code)))
	}
	return (code + 1) / 2
}

func auth(chatID int64, args []string, s *syncMap) bool {
	state := state{args[1], args[0], false}
	s.set(chatID, state)
	code := codeGenerator()
	CODES = append(CODES, code)
	email := mailCreator(chatID, s, strconv.Itoa(code))
	ch <- email
	msg := tgbotapi.NewMessage(chatID, "Ti è stata inviata una mail con il codice di verifica")
	bot.Send(msg)
	for {
		user, _ := s.get(chatID)
		if user.verified {
			log.Println("user verified" + user.ghusername)
			return true
		}
		time.Sleep(15 * time.Second)
	}
}

func mailCreator(chatID int64, s *syncMap, code string) *gomail.Message {
	user, _ := s.get(chatID)
	msg := gomail.NewMessage()
	msg.SetHeader("From", EMAIL_USERNAME)
	msg.SetHeader("To", user.address)
	msg.SetHeader("Subject", "Codice di verifica per accesso repository GitHub")
	msg.SetBody("text/html", code+" ")
	return msg
}
