package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func main() {

	srcFile := flag.String("srcFile", "", "JSON source file")

	flag.Parse()

	fmt.Println(*srcFile)

	var exportString string

	outFile := fmt.Sprintf("./output/transformed_%s", filepath.Base(*srcFile))

	// ingest JSON
	jsonFile, err := os.Open(*srcFile)

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var authors []ocaAuthor

	json.Unmarshal(byteValue, &authors)

	// New authors for new JSON
	var newAuthors []string

	// transform author data into new OMS sanity data fields

	for i := 0; i < len(authors); i++ {
		uid := uuid.New()

		slug := slug{
			Type:    "_slug",
			Current: strings.ReplaceAll(strings.ToLower(authors[i].Name), " ", "-"),
		}

		a := &author{
			Id:   ocaUUID(uid),
			Name: authors[i].Name,
			Slug: slug,
			Year: authors[i].UserRegistered[:4],
		}

		newAuthor, err := json.MarshalIndent(a, " ", "  ")
		if err != nil {
			fmt.Println(err)
		}

		newAuthors = append(newAuthors, string(newAuthor))
	}

	exportString = strings.Join(newAuthors, ",")

	// export new JSON

	exportString = fmt.Sprintf("[%s]", exportString)

	fmt.Println(exportString)
	fmt.Println(outFile)

	err = os.WriteFile(outFile, []byte(exportString), 0644)
	if err != nil {
		fmt.Println(err)
	}
}

// ocaUUID prepends `oca-` to the front of the UUID so we
// know which data objects came from OCA just by their
// ID.
func ocaUUID(gen uuid.UUID) string {
	return fmt.Sprintf("oca-%s", gen.String())
}

// Data translations

type author struct {
	Id    string `json:"_id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Slug  slug   `json:"slug"`
	Year  string `json:"year"`
}

type slug struct {
	Type    string `json:"_type"`
	Current string `json:"current"`
}

type ocaAuthor struct {
	Name           string `json:"display_name"`
	Email          string `json:"user_email"`
	UserRegistered string `json:"user_registered"`
}
