package main

import (
	"encoding/gob"
	"log"
	"os"

	postal "github.com/openvenues/gopostal/parser"
)

func main() {
	decoder := gob.NewDecoder(os.Stdin)
	encoder := gob.NewEncoder(os.Stdout)
	var request string
	for {
		err := decoder.Decode(&request)
		if err != nil {
			log.Fatalf("Error decoding - %#v", err)
		}
		encoder.Encode(postal.ParseAddress(request))
	}
}
