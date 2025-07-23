package main

import (
	. "firefly/pkg/utils"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load()

	StartServer("0.0.0.0:399201")

}
