package main

import (
	C "github.com/princjef/gomarkdoc/cmd"
	"log"
)

func main() {
	log.SetFlags(0)

	buildCmd := C.BuildCommand()

	if err := buildCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
