package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
)

// ocaUUID prepends `oca-` to the front of a given UUID so the data objects that
// came from OCA can be identified just by their ID. Also, if in the future,
// there needs to be another transition, sorting data by organization is simple.
func ocaUUID(gen uuid.UUID) string {
	return fmt.Sprintf("oca-%s", gen.String())
}

// getByteValue opens a JSON file at the given path and reads it into a byte
// array. The file is deferred to close. An error is returned if the file
// cannot be opened or read.
func getByteValue(p string) ([]byte, error) {
	jsonFile, err := os.Open(p)
	if err != nil {
		return []byte{}, err
	}

	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return []byte{}, err
	}

	return byteValue, nil
}

// Retrieved from: https://old.reddit.com/r/golang/comments/gritgv/how_to_print_nicely_a_nested_struct/
func PrintJSON(obj interface{}) {
	bytes, _ := json.MarshalIndent(obj, " ", "   ")
	fmt.Println(string(bytes))
}
