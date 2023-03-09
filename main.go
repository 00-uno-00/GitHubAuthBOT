package main

import (
	"bufio"
	"context"
	cr "crypto/rand"
	"encoding/base64"
	"errors"
	"log"
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
	//telegram vars
	bot           *tgbotapi.BotAPI
	TG_GITHUB_API = os.Getenv("GAB_TG_GITHUB_API")

	//GitHub vars
	owner               = "00-uno-00"
	Permission          = "pull"
	GITHUB_ACCESS_TOKEN = os.Getenv("GAB_GITHUB_ACCESS_TOKEN")

	//gomail env
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

type syncHash struct {
	CODES map[int64]string
	m     sync.Mutex
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

func (h *syncHash) setH(id int64, code string) {
	h.m.Lock()
	defer h.m.Unlock()
	h.CODES[id] = code
	log.Println("set code")
}

func (h *syncHash) deleteH(id int64, code string) bool {
	h.m.Lock()
	defer h.m.Unlock()
	h.CODES[id] = code
	if code == h.CODES[id] {
		delete(h.CODES, id)
		log.Println("Verified code has been removed")
		return true
	} else {
		log.Println("wrong key")
		return false
	}
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

	//syncMaps
	s := syncMap{map[int64]state{}, sync.Mutex{}}

	h := syncHash{map[int64]string{}, sync.Mutex{}}

	// Set this to true to log all interactions with telegram servers
	bot.Debug = false

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Create a new cancellable background context. Calling `cancel()` leads to the cancellation of the context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// `updates` is a golang channel which receives telegram updates
	updates := bot.GetUpdatesChan(u)

	// Pass cancellable context to goroutine
	go receiveUpdates(ctx, updates, &s, &h)

	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()
}

func receiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, s *syncMap, h *syncHash) {
	for {
		select {
		// stop looping if ctx is cancelled
		case <-ctx.Done():
			updates.Clear()
			return
		// receive update from channel and then handle it
		case update := <-updates:
			go handleMessage(ctx, update.Message, s, h)
		}
	}
}

func handleMessage(ctx context.Context, message *tgbotapi.Message, s *syncMap, h *syncHash) {

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
		hCommand(user.ID, command, args, s, h)
	} else if code := message.Text; err == nil {
		if h.deleteH(user.ID, code) {
			verifieduser, err := s.get(user.ID)
			if err != nil {
				log.Println("unexpected error")
				return
			}
			verifiedState := state{verifieduser.address, verifieduser.ghusername, true}
			s.set(user.ID, verifiedState)
		}
		return
	} else {
		sendMsg(user.ID, "invalid message")
	}

	if err != nil {
		log.Printf("An error occured: %s", err.Error())
	}
}

func hCommand(chatID int64, command string, arguments []string, s *syncMap, h *syncHash) error {
	switch command {
	case "menu":
		sendMsg(chatID, "I comandi disponibili sono: \n /verifica <username> <example@email.xyz> - verifica se puoi accedere alle repository \n /accedi <nome repo> - seleziona la repository a cui vuoi avere accesso \n /accessi - elenco degli accessi alle repository \n /gsora - colui che fornisce la VPS")
		log.Println("menu")
	case "gsora":
		sendMsg(chatID, "gsora amico delle guardie (More info at: https://noskills.club)")
	case "verifica":
		hVerifica(arguments, chatID, s, h)
	case "accessi":
		hAccessi(chatID, s)
	case "accedi":
		handleAccedi(chatID, arguments, s)
	case "aggiorna":
		handleAggiorna(arguments, chatID, s)
	case "info":
		sendMsg(chatID, "Per avere piu' info rispetto a questo bot visita: https://github.com/00-uno-00/GitHubAuthBOT")
	case "start":
		sendMsg(chatID, "Scrivi /menu per vedere i comandi disponibili")
	default:
		sendMsg(chatID, "Comando non riconosciuto")
	}

	return nil
}

func hVerifica(args []string, chatID int64, s *syncMap, h *syncHash) {
	if len(args) < 2 {
		sendMsg(chatID, "inserisci comando con username GitHub e Mail (/verifica <username> <example@email.xyz>)")
		return
	}

	user, err := s.get(chatID)
	if err != nil {
		if strings.HasSuffix(args[1], "@unito.it") || strings.HasSuffix(args[1], "@edu.unito.it") {
			go auth(chatID, args, s, h)
			return
		} else {
			sendMsg(chatID, "Email invalida")
			return
		}
	} else if user.verified {
		sendMsg(chatID, "Utente gia' verificato, per aggiornare usa /aggiorna <username> <example@email.xyz> ")
		sendMsg(chatID, "email: "+user.address)
		sendMsg(chatID, "user: "+user.ghusername)
	}
}

func handleAggiorna(args []string, chatID int64, s *syncMap) {
	if len(args) < 2 {
		sendMsg(chatID, "Inserisci comando con username GitHub e Mail (/aggiorna <username> <example@email.xyz>)")
		return
	}
	user, _ := s.get(chatID)

	if (strings.HasSuffix(args[1], "@unito.it") || strings.HasSuffix(args[1], "@edu.unito.it")) && user.verified {
		state := state{args[1], args[0], true}
		s.set(chatID, state)
		user, _ = s.get(chatID)
		sendMsg(chatID, "Dati aggiornati: "+user.ghusername)
		return
	} else if !user.verified {
		sendMsg(chatID, "verifica non effettuata")
		return
	} else {
		sendMsg(chatID, "Email invalida")
		return
	}
}

func handleAccedi(chatID int64, args []string, s *syncMap) {
	user, err := s.get(chatID)
	if !user.verified || err != nil {
		sendMsg(chatID, "verifica non effettuata")
		return
	} else if len(args) == 0 {
		rlist := strings.Join(REPOS, " \n -")
		sendMsg(chatID, "Sintassi comando: /accedi <RepoName>  Elenco delle repository: \n -"+rlist)
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
					sendMsg(chatID, "errore: "+addCollaborator(*client, user.ghusername, REPOS[i]))
					return
				}
				sendMsg(chatID, "Invito inviato")
			} else {
				sendMsg(chatID, "Accesso gia' ottenuto")
			}
			return
		}
		if i == len(REPOS)+1 {
			sendMsg(chatID, "Repo Invalida")
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
		sendMsg(chatID, "Repositories a cui hai accesso: "+repolist)
	} else if err != nil {
		sendMsg(chatID, "Utente non verificato")
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
	for m := range ch {
		var s gomail.SendCloser
		var err error
		open := false

		if !open {
			if s, err = dialer.Dial(); err != nil {
				panic(err)
			}
			open = true
		}
		if err := gomail.Send(s, m); err != nil {
			log.Print(err)
		}

	}
}

func codeGenerator() string {
	entropy := make([]byte, 8)

	if _, err := cr.Read(entropy); err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(entropy)
}

func auth(chatID int64, args []string, s *syncMap, h *syncHash) bool {
	state := state{args[1], args[0], false}
	s.set(chatID, state)

	code := codeGenerator()
	h.setH(chatID, code)

	ch <- mailCreator(chatID, s, code)

	sendMsg(chatID, "Ti è stata inviata una mail con il codice di verifica")
	for {
		user, _ := s.get(chatID)
		if user.verified {
			log.Println("user verified: " + user.ghusername)
			sendMsg(chatID, "Utente verificato")
			return true
		}
		time.Sleep(10 * time.Second)
	}
}

func mailCreator(chatID int64, s *syncMap, code string) *gomail.Message {
	user, _ := s.get(chatID)
	msg := gomail.NewMessage()
	msg.SetHeader("From", EMAIL_USERNAME)
	msg.SetHeader("To", user.address)
	msg.SetHeader("Subject", "Codice di `verifica `per accesso repository GitHub")
	msg.SetBody("text/html", code+"\nSe non hai richiesto questo codice, puoi ignorare questo messaggio. Un altro utente potrebbe avere digitato il tuo indirizzo e-mail per errore.\nPer non ricevere ulteriori mail scrivi a questo indirizzo o su telegram a https://t.me/I00uno00I")
	return msg
}

func sendMsg(chatID int64, msg string) {
	smsg := tgbotapi.NewMessage(chatID, msg)
	bot.Send(smsg)
}
