package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db *sqlx.DB

	schema = `
	CREATE TABLE IF NOT EXISTS user (
		tgID INT PRIMARY KEY,
		address TEXT,
		ghusername TEXT,
		tgusername TEXT,
		verified BOOLEAN NOT NULL DEFAULT FALSE,
		admin BOOLEAN NOT NULL DEFAULT FALSE,
		access TEXT NOT NULL DEFAULT 'pull'
		levels TEXT NOT NULL DEFAULT '[]'
		FOREIGN KEY (levels) REFERENCES level (name) ON DELETE CASCADE ON UPDATE CASCADE
		);
	
	CREATE TABLE IF NOT EXISTS repos (
		name TEXT PRIMARY KEY,
		URL TEXT,
		owner TEXT,
		lvl INT NOT NULL CHECK(lvl >= 0) DEFAULT 0);
	
	CREATE TABLE IF NOT EXISTS level (
		name TEXT PRIMARY KEY,
		repos TEXT,
		lvl INT NOT NULL CHECK (lvl >= 0) DEFAULT 0,
		FOREIGN KEY (repos) REFERENCES repos (name) ON DELETE CASCADE ON UPDATE CASCADE)
	`
)

// /DB SETUP
func setupdb() {
	db, err := sqlx.Connect("sqlite3", "file:database.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	db.MustExec(schema)
	log.Println("Database setup complete.")

}

///USER QUERIES

func insertUser(db *sqlx.DB, usr user) {
	tx := db.MustBegin()

	tx.MustExec(tx.Rebind("INSERT INTO user (tgID, address, ghusername, tgusername, verified, admin, access) VALUES (?,?,?,?,?,?,?)"), usr.tgID, usr.address, usr.ghusername, usr.tgusername, usr.verified, usr.admin, usr.access)
	tx.Commit()

	log.Println("User: " + usr.tgusername + " has been added")
}

func editUser(db *sqlx.DB, usr user) {
	tx := db.MustBegin()

	tx.MustExec(tx.Rebind("UPDATE user SET address = ?, ghusername = ?, tgusername = ?, verified = ?, admin = ?, access = ? WHERE tgID = ?"), usr.address, usr.ghusername, usr.tgusername, usr.verified, usr.admin, usr.access, usr.tgID)
	tx.Commit()

	log.Println("User: " + usr.tgusername + " has been edited")
}

func getUser(db *sqlx.DB, tgID int64) (user, error) {
	var usr user
	err := db.Get(&usr, db.Rebind("SELECT * FROM user WHERE tgID = ?"), tgID)
	if err != nil {
		return user{}, err
	}

	return usr, nil
}

func getLevels(db *sqlx.DB, tgID int64) ([]level, error) {
	var lvls []level
	err := db.Select(&lvls, db.Rebind("SELECT levels FROM user WHERE tgID = ?"), tgID)
	if err != nil {
		return []level{}, err
	}

	return lvls, nil
}

///LEVEL QUERIES

func insertLevel(db *sqlx.DB, lvl level) {
	tx := db.MustBegin()

	tx.MustExec(tx.Rebind("INSERT INTO level (name, repos, lvl) VALUES (?, ?, ?)"), lvl.name, lvl.repos, lvl.lvl)
	tx.Commit()

	log.Println("Level: " + lvl.name + " has been added")
}

func editLevel(db *sqlx.DB, lvl level) {
	tx := db.MustBegin()

	tx.MustExec(tx.Rebind("UPDATE level SET repos = ?, lvl = ? WHERE name = ?"), lvl.repos, lvl.lvl, lvl.name)
	tx.Commit()

	log.Println("Level: " + lvl.name + " has been edited")
}

func getLevel(db *sqlx.DB, name string) (level, error) {
	var lvl level
	err := db.Get(&lvl, db.Rebind("SELECT * FROM level WHERE name = ?"), name)
	if err != nil {
		return level{}, err
	}

	return lvl, nil
}

/* ask for duplicate function names
func getLevels(db *sqlx.DB, lvl int) ([]level, error) {
	var lvls []level
	err := db.Select(&lvls, db.Rebind("SELECT * FROM level WHERE lvl = ?"), lvl)
	if err != nil {
		return []level{}, err
	}

	return lvls, nil
}*/

///REPO QUERIES

func insertRepo(db *sqlx.DB, repo repository) {
	tx := db.MustBegin()

	tx.MustExec(tx.Rebind("INSERT INTO user (name, URL, owner) VALUES (?,?,?)"), repo.name, repo.URL, repo.owner)
	tx.Commit()
}

func editRepo(db *sqlx.DB, repo repository) {
	tx := db.MustBegin()

	tx.MustExec(tx.Rebind("UPDATE user SET URL = ?, owner = ? WHERE name = ?"), repo.URL, repo.owner, repo.name)
	tx.Commit()
}

func getRepo(db *sqlx.DB, name string) (repository, error) {
	var repo repository
	err := db.Get(&repo, db.Rebind("SELECT * FROM user WHERE name = ?"), name)
	if err != nil {
		return repository{}, err
	}

	return repo, nil
}
