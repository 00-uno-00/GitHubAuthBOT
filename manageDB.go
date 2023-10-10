package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db                *sql.DB
	max_conns         = 5
	lvl_conns         = make(chan lvl_query, max_conns)
	user_conns        = make(chan user_query, max_conns)
	blacklisted_users = int64(0)
)

type lvl_query struct {
	query      *sql.Stmt
	lvl        level
	query_type bool
}

type user_query struct {
	query *sql.Stmt
	user  user
}

func setupdb() {
	db, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	query, _ := db.Prepare(`
	CREATE TABLE IF NOT EXISTS user (
		tgID INT PRIMARY KEY,
		address TEXT,
		ghusername TEXT,
		tgusername TEXT,
		verified BOOLEAN NOT NULL DEFAULT FALSE,
		admin BOOLEAN NOT NULL DEFAULT FALSE,
		access TEXT NOT NULL DEFAULT 'pull')
	`)
	query.Exec()

	query, err = db.Prepare(`
		CREATE TABLE IF NOT EXISTS repos (
		name TEXT PRIMARY KEY,
		URL TEXT,
		owner TEXT,
		lvl INT NOT NULL CHECK(lvl >= 0) DEFAULT 0)
	`)
	if err != nil {
		log.Println("Error preparing query repos: ", err)
		log.Fatal(err)
	}
	query.Exec()

	query, err = db.Prepare(`
		CREATE TABLE IF NOT EXISTS level (
		name TEXT PRIMARY KEY,
		repos TEXT,
		lvl INT NOT NULL CHECK (lvl >= 0) DEFAULT 0,
		FOREIGN KEY (repos) REFERENCES repos (name) ON DELETE CASCADE ON UPDATE CASCADE)
	`)
	if err != nil {
		log.Println("Error preparing query level: ", err)
		log.Fatal(err)
	}
	query.Exec()

	log.Println("Database setup complete.")
}

func insertLevel(lvl level) {
	level, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	insert, err := db.Prepare("INSERT INTO level (name, lvl) VALUE (?,?)")
	if err != nil {
		log.Println("Error preparing query level: ", err)
		log.Fatal(err)
	}

	/*_, err = insert.Exec(lvl.name, lvl)
	if err != nil {
		log.Println("Error inserting level: ", err)
		log.Fatal(err)
	}*/

	defer func() {
		insert.Close()
		level.Close()
	}()
}

func insertUser(usr user) {
	user, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	insert, err := db.Prepare("INSERT INTO user (tgID, address, ghusername, tgusername, verified, admin, access) VALUES (?,?,?,?,?,?,?)")
	if err != nil {
		log.Println("Error preparing query user: ", err)
		log.Fatal(err)
	}

	defer func() {
		insert.Close()
		user.Close()
	}()
}

func insertRepo(repo repository) {
	repository, err := sql.Open("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	insert, err := db.Prepare("INSERT INTO user (name, URL, owner) VALUES (?,?,?)")
	if err != nil {
		log.Println("Error preparing query user: ", err)
		log.Fatal(err)
	}
	_, err = insert.Exec(repo.name, repo.URL, repo.owner)
	if err != nil {
		log.Println("Error inserting user: ", err)
		log.Fatal(err)
	}

	defer func() {
		insert.Close()
		repository.Close()
	}()
}

func execQuerylvl() { //lvl
	for query := range lvl_conns {
		switch query.query_type {
		case true: //get
			_, err := query.query.Exec(query.lvl.name)
			if err != nil {
				log.Println("Error inserting level: ", err)
				log.Fatal(err)
			}
			log.Println("lvl inserted")
		case false: //set
			_, err := query.query.Exec(query.lvl.name, query.lvl.lvl)
			if err != nil {
				log.Println("Error inserting level: ", err)
				log.Fatal(err)
			}
			log.Println("lvl inserted")
		}
	}
}

func set(query user_query) {
	_, err := query.query.Exec(query.user.tgID, query.user.address, query.user.ghusername, query.user.tgusername, query.user.verified, query.user.admin, query.user.access)
	if err != nil {
		log.Println("Error inserting user: ", err)
		log.Fatal(err)
	}
}

func verifyGithub(query user_query) {
	_, err := query.query.Exec(query.user.tgID, query.user.address, query.user.ghusername, query.user.verified)
	if err != nil {
		log.Println("Error inserting user: ", err)
		log.Fatal(err)
	}
}

func changeAccess(query user_query) {
	_, err := query.query.Exec(query.user.tgID, query.user.access)
	if err != nil {
		log.Println("Error updating user: ", err)
		log.Fatal(err)
	}
}

func adminRights(query user_query) {
	_, err := query.query.Exec(query.user.tgID, query.user.admin)
	if err != nil {
		log.Println("Error updating user: ", err)
		log.Fatal(err)
	}
}

func add_blacklist(query user_query) { //rename
	if query.user.tgID == 0 {
		blacklisted_users++
		query.user.tgID = blacklisted_users
	}
	_, err := query.query.Exec(query.user.tgID, query.user.address, query.user.ghusername, query.user.tgusername, query.user.verified, query.user.admin, query.user.access)
	if err != nil {
		log.Println("Error updating user: ", err)
		log.Fatal(err)
	}
}

func getUser(query user_query) user {
	var usr user
	err := query.query.QueryRow(query.user.tgID).Scan(&usr.tgID, &usr.address, &usr.ghusername, &usr.tgusername, &usr.verified, &usr.admin, &usr.access)
	if err != nil {
		log.Println("Error getting user: ", err)
		log.Fatal(err)
	}
	return usr
}
