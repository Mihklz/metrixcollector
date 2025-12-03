package version

import "fmt"

var (
	BuildVersion string
	BuildDate    string
	BuildCommit  string
)

// Print выводит информацию о сборке в стандартный вывод.
func Print() {
	fmt.Println("Build version:", valueOrNA(BuildVersion))
	fmt.Println("Build date:", valueOrNA(BuildDate))
	fmt.Println("Build commit:", valueOrNA(BuildCommit))
}

func valueOrNA(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}
