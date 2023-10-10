package main

import (
	"bufio"
	"context"
	cr "crypto/rand"
	"database/sql"
	"encoding/base64"
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

	REPOS = []repository{}

	ADMINS = []string{} //"I00uno00I"

	LEVELS = []level{}

	BLACKLIST = []user{}

	CODES = []int{}
)

type level struct {
	name string
	lvl  int
}

type user struct {
	tgID       int64
	address    string
	ghusername string
	tgusername string
	verified   bool
	admin      bool
	access     string
}

type repository struct {
	name  string
	URL   string
	owner string
}

type syncHash struct {
	CODES map[int64]string
	m     sync.Mutex
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

	//DB setup
	setupdb()

	//syncMaps
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
	go receiveUpdates(ctx, updates, &h)

	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()
}

func receiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, h *syncHash) {
	for {
		select {
		// stop looping if ctx is cancelled
		case <-ctx.Done():
			updates.Clear()
			return
		// receive update from channel and then handle it
		case update := <-updates:
			go handleMessage(ctx, update.Message, h)
		}
	}
}

func handleMessage(ctx context.Context, message *tgbotapi.Message, h *syncHash) {

	user := message.From
	text := message.Text

	if user == nil {
		return
	}

	// Print to console
	log.Println("handle message from:", user.UserName, ": ", text)
	var err error

	if message.IsCommand() {
		command := message.Command()
		args := strings.Fields(message.CommandArguments())
		log.Println("command:", command)
		hCommand(*user, command, args, h)
	} else if len(message.Text) == 12 {
		if h.deleteH(user.ID, message.Text) {
			verifieduser, err := ud.getData() //change
			if err != nil {
				log.Println("unexpected error")
				return
			}
			verifiedState := state{true, false, user{verifieduser.userdata.address, verifieduser.userdata.ghusername, verifieduser.userdata.tgusername, user.ID}}

			db, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
			if err != nil {
				log.Fatal(err)
			}
			query, _ := db.Prepare(`SELECT * FROM user WHERE tgID = ?`)
			usr_query := user_query{query}
			ud.setData(getUser(), verifiedState)
			return
		}
		sendMsg(user.ID, "invalid code")
	} else {
		sendMsg(user.ID, "invalid message")
	}

	if err != nil {
		log.Printf("An error occured: %s", err.Error())
	}
}

func hCommand(user tgbotapi.User, command string, arguments []string, h *syncHash) error {
	switch command {
	case "menu":
		sendMsg(user.ID, "I comandi disponibili sono: \n /verifica <username> <example@email.xyz> - verifica se puoi accedere alle repository \n /accedi <nome repo> - seleziona la repository a cui vuoi avere accesso \n /accessi - elenco degli accessi alle repository \n /gsora - colui che fornisce la VPS")
		log.Println("menu")
	case "gsora":
		sendMsg(user.ID, "gsora amico delle guardie (More info at: https://noskills.club)")
	case "verifica":
		hVerifica(arguments, user, h)
	case "accessi":
		hAccessi(user.ID)
	case "accedi":
		handleAccedi(user.ID, arguments)
	case "aggiorna":
		handleAggiorna(arguments, user.ID)
	case "info":
		sendMsg(user.ID, "Per avere piu' info rispetto a questo bot visita: https://github.com/00-uno-00/GitHubAuthBOT")
	case "start":
		sendMsg(user.ID, "Scrivi /menu per vedere i comandi disponibili")
	case "admin":
		admin_check(user)
		admin_hCommand(user.ID, arguments)
	default:
		sendMsg(user.ID, "Comando non riconosciuto")
	}

	return nil
}

func hVerifica(args []string, tguser tgbotapi.User, h *syncHash) {
	if len(args) < 2 {
		sendMsg(tguser.ID, "inserisci comando con username GitHub e Mail (/verifica <username> <example@email.xyz>)")
		return
	}

	query, err := db.Prepare(`SELECT * FROM user WHERE tgID = ?`)
	if err != nil {
		log.Println("Error preparing query user: ", err)
	}

	local_user := user{}
	query.QueryRow(tguser.ID).Scan(&local_user.tgID, &local_user.address, &local_user.tgusername, &local_user.ghusername) //to be fixed
	//query to retrieve data from db
	if err != nil {
		if strings.HasSuffix(args[1], "@unito.it") || strings.HasSuffix(args[1], "@edu.unito.it") {
			go auth(tguser, args, h)
			return
		} else {
			sendMsg(tguser.ID, "Email invalida")
			return
		}
	} else if local_user.verified {
		sendMsg(tguser.ID, "Utente gia' verificato, per aggiornare usa /aggiorna <username> <example@email.xyz> ")
		sendMsg(tguser.ID, "email: "+local_user.address)
		sendMsg(tguser.ID, "user: "+local_user.ghusername)
	}
}

func handleAggiorna(args []string, chatID int64) {
	if len(args) < 2 {
		sendMsg(chatID, "Inserisci comando con username GitHub e Mail (/aggiorna <username> <example@email.xyz>)")
		return
	}
	user, _ := s.get(chatID)

	if (strings.HasSuffix(args[1], "@unito.it") || strings.HasSuffix(args[1], "@edu.unito.it")) && user.verified {
		state := state{user.verified, user.admin, user{args[1], args[0], user.userdata.tgusername, user.userdata.tgID}}
		s.set(chatID, state)
		user, _ = s.get(chatID)
		sendMsg(chatID, "Dati aggiornati: "+user.userdata.ghusername)
		return
	} else if !user.verified {
		sendMsg(chatID, "verifica non effettuata")
		return
	} else {
		sendMsg(chatID, "Email invalida")
		return
	}
}

func handleAccedi(chatID int64, args []string) {
	user, err := s.get(chatID) //change
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
			if !checkCollaborator(&client, user.userdata.ghusername, REPOS[i]) {
				log.Println("Adding collaborator...")
				if addCollaborator(&client, user.userdata.ghusername, REPOS[i]) != "201 Created" {
					sendMsg(chatID, "errore: "+addCollaborator(&client, user.userdata.ghusername, REPOS[i]))
					return
				}
				sendMsg(chatID, "Invito inviato")
			} else {
				sendMsg(chatID, "Accesso gia' ottenuto")
			}
			return
		}
	}
	sendMsg(chatID, "Repo Invalida")
}

func hAccessi(chatID int64) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: GITHUB_ACCESS_TOKEN},
	)

	tc := oauth2.NewClient(context.Background(), ts)

	client := github.NewClient(tc)

	user, err := s.get(chatID)
	if err == nil {
		collaborates := []string{}
		for _, r := range REPOS {
			if checkCollaborator(&client, user.userdata.ghusername, r) {
				collaborates = append(collaborates, r)
			}
		}
		repolist := strings.Join(collaborates, ", ")
		sendMsg(chatID, "Repositories a cui hai accesso: "+repolist)
	} else if err != nil {
		sendMsg(chatID, "Utente non verificato")
	}
}

func checkCollaborator(client *github.Client, ghusername, RepoName string) bool {
	ret, resp, err := client.Repositories.IsCollaborator(context.Background(), owner, RepoName, ghusername)
	if err != nil {
		log.Println("errore: " + resp.Response.Status + "   " + err.Error())
	}
	return ret
}

func addCollaborator(client *github.Client, ghusername, RepoName string) string {
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

func auth(user tgbotapi.User, args []string, h *syncHash) bool {

	state := state{}
	s.set(user.ID, state)

	code := codeGenerator()
	h.setH(user.ID, code)

	ch <- mailCreator(user.ID, s, code)

	sendMsg(user.ID, "Ti è stata inviata una mail con il codice di verifica")
	for {
		local_user, _ := s.get(user.ID)
		if local_user.verified {
			log.Println("user verified: " + local_user.userdata.ghusername)
			sendMsg(user.ID, "Utente verificato")
			return true
		}
		time.Sleep(10 * time.Second)
	}
}

func mailCreator(chatID int64, code string) *gomail.Message {
	user, _ := s.get(chatID)
	msg := gomail.NewMessage()
	msg.SetHeader("From", EMAIL_USERNAME)
	msg.SetHeader("To", user.userdata.address)
	msg.SetHeader("Subject", "Codice di verifica per accesso repository GitHub")
	msg.SetBody("text/html", "<p>"+code+"</p>"+"<p><em>Se non hai richiesto questo codice, puoi ignorare questo messaggio. Un altro utente potrebbe avere digitato il tuo indirizzo e-mail per errore.\nPer non ricevere ulteriori mail scrivi a questo indirizzo o su telegram a</em> https://t.me/I00uno00I</p>")
	return msg
}

func sendMsg(chatID int64, msg string) {
	smsg := tgbotapi.NewMessage(chatID, msg)
	bot.Send(smsg)
}

func admin_check(user tgbotapi.User) {
	for i := 0; i < len(ADMINS); i++ {
		if user.UserName == ADMINS[i] {
			admin_user, err := s.get(user.ID)
			if err != nil {
				log.Println("unexpected error")
				return
			}
			admin_status := state{true, true, user{admin_user.userdata.address, admin_user.userdata.ghusername, admin_user.userdata.tgusername, admin_user.userdata.tgID}}
			s.set(user.ID, admin_status)
		}
	}
}

func blockUser(client *github.Client, ghusername string) string {
	db, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Users.BlockUser(context.Background(), ghusername)
	if err != nil {
		log.Println("errore: " + resp.Response.Status + "   " + err.Error())
		return resp.Response.Status
	}

	return resp.Response.Status
}
