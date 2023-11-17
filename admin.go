package main

import (
	"log"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func admin_hCommand(adminID int64, arguments []string, ud *syncData) {
	switch arguments[0] {
	case "addblacklist":
		addBlacklist(adminID, arguments, ud)
	case "delblacklist":
		delBlacklist(adminID, arguments[0], ud)
	case "addrepository":
		addRepository(adminID, arguments)
	case "removerepository":
		removeRepository(adminID, arguments)
	case "addlevel":
		addLevel(adminID, arguments)
	case "removelevel":
		removeLevel(adminID, arguments)
	default:
		sendMsg(adminID, "Unknown command")
	}
}

func addBlacklist(adminID int64, args []string, ud *syncData) { //args[0] = email, args[1] = ghusername, args[2] = tgusername, args[3] = tgID
	if len(args) != 4 {
		sendMsg(adminID, "Invalid number of arguments")
		return
	}
	a4, _ := strconv.ParseInt(args[4], 10, 64)
	blacklist_entry := user{tgID: a4, address: args[0], ghusername: args[1], tgusername: args[2], verified: true, admin: false, access: "blacklist"}
	sendMsg(adminID, ud.setData(args[0], blacklist_entry))
}

func delBlacklist(adminID int64, email string, ud *syncData) {
	if ud.deleteData(email) {
		sendMsg(adminID, "data deleted succefully")
	} else {
		sendMsg(adminID, "couldn't delete data")
	}
}

func addRepository(adminID int64, args []string) {
	if len(args) < 4 {
		sendMsg(adminID, "Invalid number of arguments")
		return
	}

	log.Println("Repository:" + args[2] + " has been added by")
}

func removeRepository(adminID int64, args []string) {
	if len(args) < 4 {
		sendMsg(adminID, "Invalid number of arguments")
		return
	}

	sendMsg(adminID, "Repository:"+args[2]+" has been removed")
	log.Println("Repository:" + args[2] + " has been removed")
}

func addLevel(adminID int64, args []string) { //nome  lvl
	if len(args) < 3 {
		sendMsg(adminID, "Invalid number of arguments")
		return
	}

	for _, v := range LEVELS {
		if strings.EqualFold(v.name, args[0]) {
			sendMsg(adminID, "Cannot create level with the same name")
			return
		}
	}

	ql, _, err := sq.Insert("level").Columns("name", "lvl").Values(args[0], args[1]).ToSql()

	if err != nil {
		log.Println("Error preparing query level: ", err)
		log.Fatal(err)
	}

	db.Exec(ql)

	sendMsg(adminID, "Level: "+args[0]+"has been added")
	log.Println("Level: " + args[0] + "has been added")
}

func removeLevel(adminID int64, args []string) {
	if len(args) < 3 {
		sendMsg(adminID, "Invalid number of arguments")
		return
	}

	for i, v := range LEVELS {
		if strings.EqualFold(v.name, args[0]) {
			LEVELS = append(LEVELS[:i-1], LEVELS[i+1:]...)
			sendMsg(adminID, "Level: "+args[0]+"has been removed")
			log.Println("Level: " + args[0] + "has been removed")
			return
		}
	}
}

/*
func editLevel(adminID int64, args []string) { // args[0] = attribute to be edited, args[1] = lvl name, args[2] = new attribute
	if len(args) < 3 {
		sendMsg(adminID, "Invalid number of arguments")
		return
	}

	switch args[0] {
	case "name":
		editLvlName(adminID, args[:0])
	case "repos":
		editLvlRepos(adminID, args[:0])
	case "lvl":

	case "users":

	default:
		sendMsg(adminID, "Unknown case")
	}

}*/

func editLvlName(adminID int64, args []string) {
	for _, v := range LEVELS {
		if strings.EqualFold(args[0], v.name) {
			v.name = args[1]
		}
	}
	sendMsg(adminID, "Level: "+args[0]+"has been changed to: "+args[1])
	log.Println("Level: " + args[0] + "has been changed to: " + args[1])
}

func editLvlRepos(adminID int64, args []string) {
	for _, v := range LEVELS {
		if strings.EqualFold(args[0], v.name) {
			new_lvl, _ := strconv.ParseInt(args[1], 10, 64)
			v.lvl = int(new_lvl)
		}
	}
	sendMsg(adminID, "Level: "+args[0]+"has been changed to: "+args[1])
	log.Println("Level: " + args[0] + "has been changed to: " + args[1])
}