package main

import (
	"firefly/pkg/utils"
	"log"
)

func main() {

	db := utils.OpenDatabase("database/example_test.duckdb", "ash")
	defer db.Close()

	general_channel_id, err := utils.CreateChannel(db, "general", utils.ChannelText)
	utils.Check(err)
	gaming_channel_id, err := utils.CreateChannel(db, "gaming", utils.ChannelVoice)
	utils.Check(err)

	for _, user := range utils.GetAllUsers(db) {
		log.Printf("%v", user)
	}

	utils.Check(utils.AddUserToChannel(db, "monu", utils.RoleUser, general_channel_id, "ash"))
	utils.Check(utils.AddUserToChannel(db, "monu", utils.RoleUser, gaming_channel_id, "ash"))
	// utils.Check(utils.AddUserToChannel(db, "rajesh", utils.RoleUser, general_channel_id, "monu"))

}
