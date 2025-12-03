package panicusage

import (
	"log"
	"os"
)

func okInMain() {
	// log.Fatal and os.Exit in non-main package should be reported.
	log.Fatal("fatal") // want "log.Fatal should not be called outside main.main"
	os.Exit(1)         // want "os.Exit should not be called outside main.main"
}

func panicEverywhere() {
	panic("boom") // want "use of builtin panic is forbidden"
}
