package main

import (
	. "firefly/pkg/utils"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var port = os.Getenv("PORT")
	if len(port) == 0 {
		port = "37373"
	}

	StartServer("0.0.0.0:" + port)

}
