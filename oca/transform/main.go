package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const NODE_PATH = "/run/current-system/sw/bin/node"

func main() {

	// First check if the system has node installed.
	cmd := exec.Command(NODE_PATH, "-v")
	_, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("Node not found on this system: %s", err))
	}

	config := config{
		inputs: inputs{
			ocaCategories:          "../input/oca_terms-categories.json",
			ocaUsers:               "../input/oca_users.json",
			ocaPosts:               "../input/oca_get_posts.json",
			ocaTermRelationships:   "../input/oca_term_relationships-all.json",
			ocaPostIdAndCategoryId: "../input/oca_post_id_to_category_id.json",
			ocaCategorySlugAndId:   "../input/oca_terms-categories.json",
			ocaAllTerms:            "../input/oca_terms-all.json",
		},
		outputs: outputs{
			transformedOcaUsers: "../output/transformed_oca_users.json",
		},
		js: js{
			indexJsPath:             "./js/index.js",
			inputHtmlPath:           "./js/index.html",
			outputHtmlPath:          "./js/output/output.html",
			transformedBlockContent: "./js/output/transformed_block_output.json",
		},
	}

	fmt.Println(config)

	// Phase 1: get Authors, Tags, and Categories.
	{
		byteValue, err := getByteValue(config.inputs.ocaUsers)
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

		err = os.WriteFile(config.outputs.transformedOcaTags, []byte(exportString), 0644)
		if err != nil {
			fmt.Println(err)
		}

		// Get tags.

		byteValue, err = getByteValue(config.inputs.ocaAllTerms)
		if err != nil {
			panic(err)
		}

		exportString, err = transformTags(byteValue, userMap)
		if err != nil {
			fmt.Println(err)
		}

		err = os.WriteFile(config.outputs.transformedOcaTags, []byte(exportString), 0644)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Phase 2: from the new data, assemble articles to upload.
	{

	}

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

// transformUsers transforms a data in a  `json` file into the schema of a new
// JSON file that can be converted into an `ndjson` file using the `jq` CLI
// tool.
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
			fmt.Printf("%s is a Net ID, skipping...\n", oldTags[i].Name)
			continue
		}

		// Business as usual...

		uid := uuid.New()
		document := document{
			Type: "tag",

			// No need OCA ID mark. These are just tags, and do not need
			// to equate to a point in time.
			Id: uid.String(),
		}
		slug := newSlug(oldTags[i].Slug)

		t := &tag{
			document:    document,
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

func transformArticles(byteValue []byte) (string, error) {
	var exportString string

	return exportString, nil
}

// config represents paths of input and output files.
type config struct {
	inputs  inputs
	outputs outputs
	js      js
}

type inputs struct {
	// Path of post_id to category_id JSON file.
	ocaCategories string
	ocaPosts      string

	// Path of previous OCA members.
	ocaUsers string

	ocaTermRelationships   string
	ocaPostIdAndCategoryId string
	ocaCategorySlugAndId   string
	ocaAllTerms            string

	// Path of Sanity ID JSON files.

	sanityCategoryRefIds string
	sanityTagRefIds      string
}
type outputs struct {
	// Path of the tags in Sanity JSON format.
	transformedOcaTags string

	// Path of users in Sanity JSON format.
	transformedOcaUsers string
}

// js represents paths of JS files and I/O. The value of these fields
// should mirror those of the same name in the index.js script; however,
// just with a different directory path.
//
// As an aside, these can be turned into environment variables.
type js struct {
	// Path of the main JS file, like index.js.
	indexJsPath string

	// Path of the HTML for the JS script to read from.
	inputHtmlPath string

	// Path of the transformed inputHtmlPath.
	outputHtmlPath string

	// Path of the HTML to Sanity Portable Text conversion JSON file.
	transformedBlockContent string
}

// mappings are mappings between Wordpress and Sanity data. They are then used
// to construct a Sanity article document from a Wordpress post.
type mappings struct {
	// Wordpress ID and Author Name.
	Id2Author map[int]string
	Author2Id map[string]int

	// Wordpress Author ID and Sanity Author ID. This is used to populate
	// the author reference fields of the article struct.
	AuthorRef2Id map[string]int
	Id2AuthorRef map[int]string

	// Image asset ID on Sanity and the Wordpress image path. This is used to
	// create image asset references for portable text content.
	SanityImageId2WordpressPath map[string]string
	WordpressPath2SanityImageId map[string]string

	// Tag slug and the corresponding Tag's Sanity document ID. This is used to
	// populate the tag reference fields of the article struct.
	TagSlug2SanityTagId map[string]string
	SanityTagId2TagSlug map[string]string

	// Slug of the Wordpress category to the Sanity category ID.
	CategorySlug2SanityCategoryId map[string]string
	SanityCategoryId2CategorySlug map[string]string

	// Wordpress
	WordpressPostId2CategorySlug map[int]string
	CategorySlug2WordpressPostId map[string][]int

	// Sanity Category that maps to a Wordpress Category. Because some
	// categories on Wordpress do not exist on Sanity, an array of strings
	// is used to represent the new associations.
	SanityCategorySlug2WordpressCategorySlug map[string][]string
	WordpressCategorySlug2SanityCategorySlug map[string]string

	WordpressId2WordpressCategory map[int]int
	WordpressCategory2WordpressId map[int][]int

	CategoryId2CategorySlug map[int]string
	CategorySlug2CategoryId map[string]int
}

func newMappings(c config) (*mappings, error) {
	mappings := &mappings{
		Id2Author: make(map[int]string),
		Author2Id: make(map[string]int),

		AuthorRef2Id: make(map[string]int),
		Id2AuthorRef: make(map[int]string),

		SanityImageId2WordpressPath: make(map[string]string),
		WordpressPath2SanityImageId: make(map[string]string),

		TagSlug2SanityTagId: map[string]string{
			// The values here are populated by importing JSON data.
			"china-global":           "",
			"business-and-economics": "",
			"events":                 "",
			"uncategorized":          "",
			"how-to-get-an-a":        "",
			"campus-life":            "",
			"student-government":     "",
			"multilingual":           "",
			"food":                   "",
			"off-campus":             "",
			"fashion":                "",
			"on-century-avenue":      "",
		},
		SanityTagId2TagSlug: map[string]string{},

		CategorySlug2SanityCategoryId: make(map[string]string),
		SanityCategoryId2CategorySlug: make(map[string]string),

		// Remap the OCA Article's category to either a Sanity category or a
		// Sanity tag, because some OCA Articles are categorized under
		// categories that no longer exist on Sanity.
		SanityCategorySlug2WordpressCategorySlug: map[string][]string{
			"news": {
				"news",
				"chineseglobal",
				"business-and-economics",
				"events",
				"uncategorized",
			},
			"opinion": {
				"opinion",
				"how-to-get-an-a",
			},
			"people": {
				"campus-life",
				"student-government",
				"multilingual",
			},
			"culture": {
				"culture",
				"food-and-nightlife",
				"lifestyle",
				"off-campus",
				"fashion",
			},
			"multimedia": {},
		},
		WordpressCategorySlug2SanityCategorySlug: map[string]string{
			// News
			"news":                   "news",
			"uncategorized":          "news",
			"chineseglobal":          "news",
			"business-and-economics": "news",
			"events":                 "news",

			// Opinion
			"opinion":         "opinion",
			"how-to-get-an-a": "opinion",

			// People
			"campus-life":        "people",
			"student-government": "people",
			"multilingual":       "people",

			// Culture
			"culture":            "culture",
			"food-and-nightlife": "culture",
			"lifestyle":          "culture",
			"off-campus":         "culture",
			"fashion":            "culture",
		},

		WordpressId2WordpressCategory: make(map[int]int),
		WordpressCategory2WordpressId: make(map[int][]int),

		CategoryId2CategorySlug: make(map[int]string),
		CategorySlug2CategoryId: make(map[string]int),
	}

	// Populate the Wordpress ID to Wordpress Category maps.

	byteValue, err := getByteValue(c.inputs.ocaPostIdAndCategoryId)
	if err != nil {
		return nil, err
	}

	d := []struct {
		PostId     int `json:"post_id"`
		CategoryId int `json:"category_id"`
	}{}

	err = json.Unmarshal(byteValue, &d)
	if err != nil {
		return nil, err
	}

	for _, v := range d {
		mappings.WordpressId2WordpressCategory[v.PostId] = v.CategoryId

		_, ok := mappings.WordpressCategory2WordpressId[v.CategoryId]

		if !ok {
			mappings.WordpressCategory2WordpressId[v.CategoryId] = []int{v.PostId}
		} else {
			mappings.WordpressCategory2WordpressId[v.CategoryId] = append(mappings.WordpressCategory2WordpressId[v.CategoryId], v.PostId)
		}
	}

	// Populate the Category ID to Category Slug maps.

	byteValue, err = getByteValue(c.inputs.ocaPostIdAndCategoryId)
	if err != nil {
		return nil, err
	}

	f := []struct {
		TermId int    `json:"term_id"`
		Slug   string `json:"slug"`
	}{}

	err = json.Unmarshal(byteValue, &f)
	if err != nil {
		return nil, err
	}

	for _, v := range f {
		mappings.CategoryId2CategorySlug[v.TermId] = v.Slug
		mappings.CategorySlug2CategoryId[v.Slug] = v.TermId
	}

	return mappings, nil
}

// Data translations

// Document represents data that all Sanity documents must have.
type document struct {
	Type string `json:"_type"`
	Id   string `json:"_id,omitempty"`
	Key  string `json:"_key,omitempty"`
}

type slug struct {
	Type    string `json:"_type"`
	Current string `json:"current"`
}

func newSlug(s string) slug {
	return slug{
		Type:    "_slug",
		Current: s,
	}
}

type author struct {
	document
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Slug  slug   `json:"slug"`
	Year  int    `json:"year"`

	// NetID is not needed for the export, however it is useful when using the
	// output data to sort through other JSON files. For example, removing
	// unnecessary tags that are NetIDs from the Wordpress dataset.
	NetID string `json:"netid,omitempty"`
}

func newAuthor(a ocaAuthor) (*author, error) {
	uid := uuid.New()
	document := document{
		Type: "member",
		Id:   ocaUUID(uid),
	}
	slug := newSlug(strings.ReplaceAll(strings.ToLower(a.Name), " ", "-"))

	year, err := strconv.Atoi(a.UserRegistered[:4])
	if err != nil {
		return nil, err
	}

	newAuthor := &author{
		document: document,
		Name:     a.Name,
		Slug:     slug,
		Year:     year,
		NetID:    a.UserLogin,
	}

	return newAuthor, nil
}

type tag struct {
	document
	Slug        slug   `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type refAuthor struct {
	document

	// Ref is the `_id` of the Member document.
	// See: https://www.sanity.io/docs/reference-type#e97572ca6050
	Ref string `json:"_ref"`
}

type refCategory struct {
	document

	// Ref is the `_id` of the Category document.
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

type refTag struct {
	document

	// Ref is the `_id` of the Tag document.
	// See: https://www.sanity.io/docs/reference-type#e97572ca6050
	Ref string `json:"_ref"`
}

func newRefTag(ref string) refTag {
	return refTag{
		document: document{
			Type: "tag",
		},
		Ref: ref,
	}
}

// ocaAuthor mirrors the Wordpress data of an author.
type ocaAuthor struct {
	// Id is used to find associated post.
	Id    int    `json:"ID"`
	Name  string `json:"display_name"`
	Email string `json:"user_email"`

	// UserRegistered is the date the user registered on Wordpress.
	UserRegistered string `json:"user_registered"`

	// UserLogin is the NetID of the user.
	UserLogin string `json:"user_login"`
}

// ocaTag mirrors the Wordpress data of a tag.
type ocaTag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ocaArticle mirrors the Wordpress data of an article.
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

type ocaCategory struct {
	Id   int    `json:"term_id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
