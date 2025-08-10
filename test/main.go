package main

import (
	"os"
	"regexp"
	"strings"
)

// (?is)create.*?\(.*?\);

func splitCreate(s string) []string {
	re := regexp.MustCompile(`(?is)create.*?\(.*?\);`)
	return re.FindAllString(s, -1)
}

func writeFile(filename string, data []string) error {
	return os.WriteFile(filename, []byte(strings.Join(data, "\n")), 0644)
}

func readFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	value := string(data)
	re := regexp.MustCompile(` {2,}`)
	result := re.ReplaceAllString(value, " ")
	re = regexp.MustCompile(`\n{2,} {2,}`)
	result = re.ReplaceAllString(result, "\n")
	return result, nil
}

func createTemp() {
	os.MkdirAll("temp", 0755)
}

func deleteTemp() {
	os.RemoveAll("temp")
}

func main() {
	createTemp()
	// defer deleteTemp()
	args := os.Args
	if len(args) < 2 {
		println("Usage: go run main.go <schema_file>")
		return
	}
	schema_file1, err := readFile(args[1])
	if err != nil {
		panic(err)
	}
	schema_file2, err := readFile(args[2])
	if err != nil {
		panic(err)
	}

	schema1 := splitCreate(schema_file1)
	schema2 := splitCreate(schema_file2)

	writeFile("temp/schema1.sql", schema1)
	writeFile("temp/schema2.sql", schema2)

}
