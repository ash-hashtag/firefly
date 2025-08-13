package main

import (
	. "firefly/pkg/utils"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load()

	var port = os.Getenv("PORT")
	if len(port) == 0 {
		port = "37373"
	}

	StartServer("0.0.0.0:" + port)

}
