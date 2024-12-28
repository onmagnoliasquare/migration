package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func main() {

	srcFile := flag.String("srcFile", "", "JSON source file")

	flag.Parse()

	// Map of users from OMS json.

	// Destination file.
	outFile := fmt.Sprintf("./output/transformed_%s", filepath.Base(*srcFile))

	byteValue, err := getByteValue(*srcFile)
	if err != nil {
		panic(err)
	}

	userMap, err := makeUserMap(byteValue)
	if err != nil {
		panic(err)
	}

	exportString, err := transformUsers(byteValue)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(outFile)

	err = os.WriteFile(outFile, []byte(exportString), 0644)
	if err != nil {
		fmt.Println(err)
	}

	byteValue, err = getByteValue("../output/oca_terms-all.json")
	if err != nil {
		panic(err)
	}

	exportString, err = transformTags(byteValue, userMap)
	if err != nil {
		fmt.Println(err)
	}

	outFile = "./output/transformed_oca_tags.json"

	fmt.Println(outFile)

	err = os.WriteFile(outFile, []byte(exportString), 0644)
	if err != nil {
		fmt.Println(err)
	}
}

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

func makeUserMap(byteValue []byte) (map[string]bool, error) {
	var authors []ocaAuthor
	userMap := make(map[string]bool)

	err := json.Unmarshal(byteValue, &authors)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(authors); i++ {
		userMap[authors[i].UserLogin] = true
	}

	return userMap, nil
}

// transformUsers transforms a `json` file into a new JSON file that can be
// converted into an `ndjson` file using the `jq` CLI tool.
func transformUsers(byteValue []byte) (string, error) {
	var authors []ocaAuthor
	var exportString string
	var newAuthors []string

	err := json.Unmarshal(byteValue, &authors)
	if err != nil {
		return "", err
	}

	// Transform author data into new OMS sanity data fields
	for i := 0; i < len(authors); i++ {
		a, err := newAuthor(authors[i])
		if err != nil {
			return "", err
		}

		newAuthor, err := json.MarshalIndent(a, " ", "  ")
		if err != nil {
			return "", err
		}

		newAuthors = append(newAuthors, string(newAuthor))
	}

	exportString = strings.Join(newAuthors, ",")
	exportString = fmt.Sprintf("[%s]", exportString)

	return exportString, nil
}

func transformTags(byteValue []byte, um map[string]bool) (string, error) {
	var oldTags []ocaTag
	var exportString string
	var newTags []string

	json.Unmarshal(byteValue, &oldTags)

	for i := 0; i < len(oldTags); i++ {
		// First check if this tag is a NetID. If so, skip it.
		if um[oldTags[i].Name] {
			continue
		}

		// Business as usual...

		uid := uuid.New()
		document := Document{
			Type: "tag",

			// No need OCA ID mark. These are just tags, and do not need
			// to equate to a point in time.
			Id: uid.String(),
		}
		slug := newSlug(oldTags[i].Slug)

		t := &tag{
			Document:    document,
			Slug:        slug,
			Name:        strings.ToLower(oldTags[i].Name),
			Description: "A tag from OCA.",
		}

		newTag, err := json.MarshalIndent(t, " ", "  ")
		if err != nil {
			return "", err
		}

		newTags = append(newTags, string(newTag))
	}

	exportString = strings.Join(newTags, ",")
	exportString = fmt.Sprintf("[%s]", exportString)

	return exportString, nil
}

// ocaUUID prepends `oca-` to the front of the UUID so we know which data
// objects came from OCA just by their ID. This also helps so that in the
// future, if there needs to be another transition, sorting data by
// organization is easier.
func ocaUUID(gen uuid.UUID) string {
	return fmt.Sprintf("oca-%s", gen.String())
}

// Data translations

// Document represents data that all Sanity documents must have.
type Document struct {
	Type string `json:"_type"`
	Id   string `json:"_id"`
}

type Slug struct {
	Type    string `json:"_type"`
	Current string `json:"current"`
}

func newSlug(s string) Slug {
	return Slug{
		Type:    "_slug",
		Current: s,
	}
}

type author struct {
	Document
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Slug  Slug   `json:"slug"`
	Year  int    `json:"year"`

	// NetID is not needed for the export, however it is useful when using the
	// output data to sort through other JSON files. For example, removing
	// unnecessary tags that are NetIDs from the Wordpress dataset.
	NetID string `json:"netid,omitempty"`
}

func newAuthor(a ocaAuthor) (*author, error) {
	uid := uuid.New()
	document := Document{
		Type: "member",
		Id:   ocaUUID(uid),
	}
	slug := newSlug(strings.ReplaceAll(strings.ToLower(a.Name), " ", "-"))

	year, err := strconv.Atoi(a.UserRegistered[:4])
	if err != nil {
		return nil, err
	}

	newAuthor := &author{
		Document: document,
		Name:     a.Name,
		Slug:     slug,
		Year:     year,
		NetID:    a.UserLogin,
	}

	return newAuthor, nil
}

type tag struct {
	Document
	Slug        Slug   `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type article struct {
	document
	Slug         slug        `json:"slug"`
	Title        string      `json:"title"`
	Subtitle     string      `json:"subtitle"`
	Abstract     string      `json:"abstract"`
	Authors      []author    `json:"authors"`
	Date         string      `json:"date"`
	CreatedAt    string      `json:"_createdAt"`
	UpdatedAt    string      `json:"_updatedAt"`
	Content      content     `json:"content"`
	Tags         []refTag    `json:"tags"`
	Category     refCategory `json:"category"`
	UseCustomCss bool        `json:"useCustomCss"`
}

type content struct{}

type refCategory struct {
	document

	// Ref is the `_id` of the Category document.
	// See: https://www.sanity.io/docs/reference-type#e97572ca6050
	Ref string `json:"_ref"`
}

type refTag struct {
	document

	// Ref is the `_id` of the Tag document.
	// See: https://www.sanity.io/docs/reference-type#e97572ca6050
	Ref string `json:"_ref"`
}

func newRefCategory(ref string) refCategory {
	return refCategory{
		document: document{
			Type: "reference",
		},
		Ref: ref,
	}
}

type ocaAuthor struct {
	Name  string `json:"display_name"`
	Email string `json:"user_email"`

	// UserRegistered is the date the user registered on Wordpress.
	UserRegistered string `json:"user_registered"`

	// UserLogin is the NetID of the user.
	UserLogin string `json:"user_login"`
}

type ocaTag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ocaArticle struct {
	Id              int    `json:"ID"`
	PostAuthor      int    `json:"post_author"`
	PostDateGMT     string `json:"post_date_gmt"`
	PostModifiedGMT string `json:"post_modified_gmt"`
	PostContent     string `json:"post_content"`
	PostTitle       string `json:"post_title"`
	PostExcerpt     string `json:"post_excerpt"`
	PostName        string `json:"post_name"`
}
