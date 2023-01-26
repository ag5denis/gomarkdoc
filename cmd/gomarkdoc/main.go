package main

import (
	C "github.com/ag5denis/gomarkdoc/cmd"
	"log"
)

func main() {
	log.SetFlags(0)

	buildCmd := C.BuildCommand()

	if err := buildCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
