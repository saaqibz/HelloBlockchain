package main

import (
	"log"

	"example.com/blockchain/src/pow"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go pow.POW()
	log.Fatal(pow.RunServer())
}
