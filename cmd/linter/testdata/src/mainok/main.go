package main

import (
	"log"
	"os"
)

// В этом пакете log.Fatal и os.Exit используются только в main.main
// и не должны порождать диагностик.
func main() {
	log.Fatal("fatal in main")
	os.Exit(1)
}

func helper() {
	// здесь вызовов нет, чтобы убедиться, что анализатор не даёт ложных срабатываний
}
