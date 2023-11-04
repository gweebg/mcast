package bootstrap

import (
	"encoding/json"
	"log"
	"net"
	"os"
)

func MustParse(filename string) NeighbourDatabase {

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not read file '%s'\n", filename)
	}

	var db NeighbourDatabase
	err = json.Unmarshal(data, &db)
	if err != nil {
		log.Fatal(err)
	}

	for key, _ := range db {
		val := net.ParseIP(key)
		if val == nil {
			log.Fatalf("| invalid IP address: %v\n", key)
		}
	}

	return db
}
