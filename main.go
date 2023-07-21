package main

import (
	"bufio"
	"context"
	cr "crypto/rand"
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

type syncData struct {
	data map[string]user //key = email
	m    sync.Mutex
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

func (ud *syncData) setData(email string, nd user) string {
	ud.m.Lock() //???
	defer ud.m.Unlock()

	ud.data[email] = nd
	log.Println("set UserData")
	return "set UserData"
}

func (ud *syncData) getData(email string) (user, bool) {
	ud.m.Lock()
	defer ud.m.Unlock()
	if data, ok := ud.data[email]; ok {
		return data, ok
	}
	return user{}, false
}

func (ud *syncData) deleteData(email string) bool {
	ud.m.Lock()
	defer ud.m.Unlock()
	if _, ok := ud.getData(email); ok {
		delete(ud.data, email)
		return true
	}
	return false
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

	ud := syncData{map[string]user{}, sync.Mutex{}}

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
	go receiveUpdates(ctx, updates, &s, &h, &ud)

	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()
}

func receiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, s *syncMap, h *syncHash, ud *syncData) {
	for {
		select {
		// stop looping if ctx is cancelled
		case <-ctx.Done():
			updates.Clear()
			return
		// receive update from channel and then handle it
		case update := <-updates:
			go handleMessage(ctx, update.Message, s, h, ud)
		}
	}
}

func handleMessage(ctx context.Context, message *tgbotapi.Message, s *syncMap, h *syncHash, ud *syncData) {

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
		hCommand(*user, command, args, s, h, ud)
	} else if len(message.Text) == 12 {
		if h.deleteH(user.ID, message.Text) {
			verifieduser, err := s.get(user.ID)
			if err != nil {
				log.Println("unexpected error")
				return
			}
			verifiedState := state{true, false, user{verifieduser.userdata.address, verifieduser.userdata.ghusername, verifieduser.userdata.tgusername, user.ID}}
			s.set(user.ID, verifiedState)
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

func hCommand(user tgbotapi.User, command string, arguments []string, s *syncMap, h *syncHash, ud *syncData) error {
	switch command {
	case "menu":
		sendMsg(user.ID, "I comandi disponibili sono: \n /verifica <username> <example@email.xyz> - verifica se puoi accedere alle repository \n /accedi <nome repo> - seleziona la repository a cui vuoi avere accesso \n /accessi - elenco degli accessi alle repository \n /gsora - colui che fornisce la VPS")
		log.Println("menu")
	case "gsora":
		sendMsg(user.ID, "gsora amico delle guardie (More info at: https://noskills.club)")
	case "verifica":
		hVerifica(arguments, user, s, h)
	case "accessi":
		hAccessi(user.ID, s)
	case "accedi":
		handleAccedi(user.ID, arguments, s)
	case "aggiorna":
		handleAggiorna(arguments, user.ID, s)
	case "info":
		sendMsg(user.ID, "Per avere piu' info rispetto a questo bot visita: https://github.com/00-uno-00/GitHubAuthBOT")
	case "start":
		sendMsg(user.ID, "Scrivi /menu per vedere i comandi disponibili")
	case "admin":
		admin_check(user, s)
		admin_hCommand(user.ID, arguments, ud)
	default:
		sendMsg(user.ID, "Comando non riconosciuto")
	}

	return nil
}

func hVerifica(args []string, user tgbotapi.User, s *syncMap, h *syncHash) {
	if len(args) < 2 {
		sendMsg(user.ID, "inserisci comando con username GitHub e Mail (/verifica <username> <example@email.xyz>)")
		return
	}

	local_user, err := s.get(user.ID)
	if err != nil {
		if strings.HasSuffix(args[1], "@unito.it") || strings.HasSuffix(args[1], "@edu.unito.it") {
			go auth(user, args, s, h)
			return
		} else {
			sendMsg(user.ID, "Email invalida")
			return
		}
	} else if local_user.verified {
		sendMsg(user.ID, "Utente gia' verificato, per aggiornare usa /aggiorna <username> <example@email.xyz> ")
		sendMsg(user.ID, "email: "+local_user.userdata.address)
		sendMsg(user.ID, "user: "+local_user.userdata.ghusername)
	}
}

func handleAggiorna(args []string, chatID int64, s *syncMap) {
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
			if !checkCollaborator(*client, user.userdata.ghusername, REPOS[i]) {
				log.Println("Adding collaborator...")
				if addCollaborator(*client, user.userdata.ghusername, REPOS[i]) != "201 Created" {
					sendMsg(chatID, "errore: "+addCollaborator(*client, user.userdata.ghusername, REPOS[i]))
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
			if checkCollaborator(*client, user.userdata.ghusername, r) {
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

func auth(user tgbotapi.User, args []string, s *syncMap, h *syncHash) bool {

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

func mailCreator(chatID int64, s *syncMap, code string) *gomail.Message {
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

func admin_check(user tgbotapi.User, s *syncMap) {
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

func blockUser(client github.Client, ghusername) {
	db, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Users.BlockUser(context.Background(), ghusername)
	if err != nil {
		log.Println("errore: " + resp.Response.Status + "   " + err.Error())
		return resp.Response.Status
	}
	
	result, err := db.Prepare("SELECT * FROM repos")
	var repo string
	result.

	return resp.Response.Status
}
