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

const NODE_PATH = "/opt/homebrew/bin/node"

func main() {

	// First check if the system has node installed.
	cmd := exec.Command(NODE_PATH, "-v")
	_, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("Node not found on this system: %s", err))
	}

	config := config{
		inputs: inputs{
			ocaUsers: "../input/oca_users.json",
			// ocaPosts:                 "../input/oca_get_posts.json",
			ocaPosts:                 "../input/oca_article_test_1.json",
			ocaTermRelationships:     "../input/oca_term_relationships-all.json",
			ocaPostIdAndCategoryId:   "../input/oca_post_id_to_category_id.json",
			ocaCategorySlugAndId:     "../input/oca_terms-categories.json",
			ocaAllTerms:              "../input/oca_terms-all.json",
			ocaPostIdAndTagSlug:      "../input/oca_post_id_and_tag_slug.json",
			sanityTagSlugAndRef:      "../input/sanity_tag_slug_and_ref.json",
			sanityMediaNameAndRefs:   "../input/sanity_media_name_and_refs.json",
			sanityCategorySlugsAndId: "../input/sanity_category_slugs_and_ids.json",
			sanityAuthorIds:          "../input/sanity_author_ids.json",
		},
		outputs: outputs{
			transformedOcaUsers:    "../output/transformed_oca_users.json",
			transformedOcaTags:     "../output/transformed_oca_tags.json",
			transformedOcaArticles: "../output/transformed_oca_articles.json",
		},
		js: js{
			indexJsPath:            "./js/index.js",
			inputHtmlPath:          "./js/index.html",
			outputHtmlPath:         "./js/output/output.html",
			transformedBlockOutput: "./js/output/transformed_block_output.json",
		},
	}

	var mappings *mappings

	// Phase 0: Generate mappings based on config.
	{
		fmt.Println()
		fmt.Println("=====================")
		fmt.Println("= BEGINNING PHASE 0 =")
		fmt.Println("=====================")

		mappings, err = newMappings(config)
		if err != nil {
			panic(err)
		}
	}

	// Phase 1: get Authors, Tags, and Categories.
	{
		fmt.Println()
		fmt.Println("=====================")
		fmt.Println("= BEGINNING PHASE 1 =")
		fmt.Println("=====================")

		fmt.Println("Reading OCA users...")

		byteValue, err := getByteValue(config.inputs.ocaUsers)
		if err != nil {
			panic(err)
		}

		fmt.Println("Attempting to transform users...")

		exportString, err := transformUsers(byteValue)
		if err != nil {
			fmt.Println(err)
		}

		transformedOcaUsersPath := config.outputs.transformedOcaUsers

		err = os.WriteFile(transformedOcaUsersPath, []byte(exportString), 0644)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("  - User transforms written to: %s\n", transformedOcaUsersPath)

		// Get tags.

		fmt.Println("Reading OCA tags...")

		byteValue, err = getByteValue(config.inputs.ocaAllTerms)
		if err != nil {
			panic(err)
		}

		fmt.Println("Attempting to transform tags...")

		exportString, err = transformTags(byteValue, mappings.authorMap)
		if err != nil {
			fmt.Println(err)
		}

		transformedTagsPath := config.outputs.transformedOcaTags

		err = os.WriteFile(transformedTagsPath, []byte(exportString), 0644)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("  - Tag transforms written to: %s\n", transformedTagsPath)
	}

	// Phase 1.5: Upload The Authors and Tags to Sanity, along with media like
	// images. This should be done before Phase 2 begins. This phase is semi-
	// automatic and is not executed by this program.

	// Phase 2: from the new data, assemble articles to upload.
	{
		fmt.Println()
		fmt.Println("=====================")
		fmt.Println("= BEGINNING PHASE 2 =")
		fmt.Println("=====================")

		ocaPostsPath := config.inputs.ocaPosts

		byteValue, err := getByteValue(ocaPostsPath)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Retrieved posts from: %s\n", ocaPostsPath)

		fmt.Println("Attempting to transform articles...")

		exportString, err := transformArticles(byteValue, *mappings, config)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Articles transformed successfully.")

		articleOutputPath := config.outputs.transformedOcaArticles

		err = os.WriteFile(articleOutputPath, []byte(exportString), 0644)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("  - Article transforms written to: %s\n", articleOutputPath)
	}

}

// transformUsers transforms data in a `json` file into the schema of a new
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
			// fmt.Printf("%s is a Net ID, skipping...\n", oldTags[i].Name)
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

func transformArticles(byteValue []byte, m mappings, c config) (string, error) {
	var ocaArticles []ocaArticle
	var exportString string
	var newArticles []string

	json.Unmarshal(byteValue, &ocaArticles)

	for _, v := range ocaArticles {
		article, err := newArticle(v, m, c)
		if err != nil {
			return "", err
		}

		newArticle, err := json.MarshalIndent(article, " ", "  ")
		if err != nil {
			return "", err
		}

		newArticles = append(newArticles, string(newArticle))
	}

	exportString = strings.Join(newArticles, ",")
	exportString = fmt.Sprintf("[%s]", exportString)

	return exportString, nil
}

// config represents paths of input and output files.
type config struct {
	inputs  inputs
	outputs outputs
	js      js
}

type inputs struct {
	// ocaPosts is a path to a JSON file containing multiple attributes related
	// to all written text published posts, and not drafts.
	//
	// SQL Query:
	//
	// SELECT *
	// FROM wp_x7zvdw3xj9_posts
	// WHERE post_parent = 0 AND post_type = 'post' AND post_status = 'publish'
	// ORDER BY post_date;
	ocaPosts string

	// ocaUsers is a path to a JSON file containing multiple attributes related
	// to previous user data.
	//
	// SQL Query:
	//
	// SELECT t.* FROM db.wp_x7zvdw3xj9_users t;
	ocaUsers string

	ocaTermRelationships   string
	ocaPostIdAndCategoryId string

	// ocaCategorySlugAndId is a path to a JSON file containing attributes
	// related to the several main categories from the Wordpress database.
	//
	// SQL Query:
	//
	// SELECT *
	// FROM db.wp_x7zvdw3xj9_terms
	// WHERE term_id IN (1,17,84,85,86,91,92,93,94,95,122,130,525,570,716,939);
	ocaCategorySlugAndId string

	ocaAllTerms string

	// ocaPostIdAndTagSlug is a path to a JSON file containing an attribute
	// "post_id" and an attribute  "tag_slug". The post IDs are IDs of posts
	// that have a certain tag. This file also excludes the initial categories,
	// only including the terms that aren't categories.
	//
	// SQL Query:
	//
	// SELECT
	//     p.ID AS post_id,
	//     tr.term_taxonomy_id AS category_id,
	//     tt.term_id AS term_id,
	//     t.slug as tag_slug
	// FROM
	//     db.wp_x7zvdw3xj9_posts p
	// JOIN
	//     db.wp_x7zvdw3xj9_term_relationships tr ON p.ID = tr.object_id
	// JOIN
	//     db.wp_x7zvdw3xj9_term_taxonomy tt ON tr.term_taxonomy_id = tt.term_taxonomy_id
	// JOIN
	//     db.wp_x7zvdw3xj9_terms t ON tt.term_id = t.term_id
	// WHERE
	//     p.post_type = 'post'
	//     AND p.post_status = 'publish'
	//     AND tr.term_taxonomy_id NOT IN (1, 17, 84, 85, 86, 91, 92, 93, 94, 95, 122, 130, 525, 570, 716, 939);
	ocaPostIdAndTagSlug string

	// Path of Sanity ID JSON files.

	// A JSON file of the categories on Sanity and their corresponding IDs.
	//
	// Sanity Query:
	//
	// *[_type == "category"]{
	// 		"refId": _id,
	//  	"slug": slug.current
	// }
	sanityCategorySlugsAndId string

	// A JSON file of every Sanity tag with attributes for their slug and
	// their ID.
	//
	// Sanity Query:
	//
	// *[_type == "tag"]{
	// 		"refId": _id,
	// 		"slug": slug.current
	// }
	sanityTagSlugAndRef string

	// A JSON file containing all Sanity media's file names and IDs.
	//
	// Sanity Query:
	//
	// *[_type == "sanity.imageAsset" && originalFilename == "oca-2014-02-IMG_0594.png"]{
	// 		"file": originalFilename,
	// 		assetId,
	// 		"refId": _id
	// }
	sanityMediaNameAndRefs string

	// A JSON file containing all Sanity member names and IDs.
	//
	// Sanity Query:
	//
	// *[_type == "member"] {
	//   name,
	//   _id
	// }
	sanityAuthorIds string
}

type outputs struct {
	// Path of the tags in Sanity JSON format.
	transformedOcaTags string

	// Path of users in Sanity JSON format.
	transformedOcaUsers string

	// Path of articles in Sanity JSON format.
	transformedOcaArticles string
}

// js represents paths of JS files and I/O. The file in of these fields
// should mirror those of the same name in the index.js script. That path to the
// files though, are just a different directory path, relative to where this
// program is run.
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
	transformedBlockOutput string
}

// mappings are mappings between Wordpress and Sanity data. They are then used
// to construct a Sanity article document from a Wordpress post.
type mappings struct {
	// Map of all authors. This field is used to check whether an author
	// exists in the database.
	authorMap map[string]bool

	// Map of all current Sanity categories. This field is used to check
	// whether a particular category exists in Sanity.
	SanityCategories map[string]bool

	// Wordpress ID and Author Name.
	AuthorId2AuthorName map[int]string
	AuthorName2AuthorId map[string]int

	// Wordpress Author ID and Sanity Author ID. This is used to populate
	// the author reference fields of the article struct.
	SanityAuthorRef2AuthorName map[string]string
	AuthorName2SanityAuthorRef map[string]string

	// Image asset ID on Sanity and the Wordpress image path. This is used to
	// create image asset references for portable text content.
	SanityImageId2WordpressPath map[string]string
	WordpressPath2SanityImageId map[string]string

	// One-to-one relationship between an image name on Sanity and its ID.
	// The image name should be of the form YYYY-MM-{name}.webp. This mapping
	// is used when linking a Sanity reference ID to a Wordpress image.
	SanityImageFilename2SanityImageId map[string]string
	SanityImageId2SanityImageFilename map[string]string

	// Tag slug and the corresponding Tag's Sanity document ID. This is used to
	// populate the tag reference fields of the article struct. This data is
	// retrieved from a Sanity query.
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

	// ID of the Wordpress post to the ID of its category.
	WordpressId2WordpressCategoryId map[int]int
	WordpressCategoryId2WordpressId map[int][]int

	CategoryId2CategorySlug map[int]string
	CategorySlug2CategoryId map[string]int

	WordpressPostId2TagSlug map[string]string
	TagSlug2WordpressPostId map[string]string

	// Wordpress posts and their tags.
	WordpressPostId2TagSlugs map[int][]string
	TagSlugs2WordpressPostId map[string][]int
}

func newMappings(config config) (*mappings, error) {
	mappings := &mappings{
		authorMap: make(map[string]bool),

		SanityCategories: map[string]bool{
			"news":    true,
			"opinion": true,
			"people":  true,
			"culture": true,
		},

		AuthorId2AuthorName: make(map[int]string),
		AuthorName2AuthorId: make(map[string]int),

		SanityAuthorRef2AuthorName: make(map[string]string),
		AuthorName2SanityAuthorRef: make(map[string]string),

		SanityImageId2WordpressPath: make(map[string]string),
		WordpressPath2SanityImageId: make(map[string]string),

		SanityImageFilename2SanityImageId: make(map[string]string),
		SanityImageId2SanityImageFilename: make(map[string]string),

		TagSlug2SanityTagId: map[string]string{
			// The values here are populated by importing JSON data.
			"chineseglobal":          "",
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

		WordpressId2WordpressCategoryId: make(map[int]int),
		WordpressCategoryId2WordpressId: make(map[int][]int),

		CategoryId2CategorySlug: make(map[int]string),
		CategorySlug2CategoryId: make(map[string]int),

		WordpressPostId2TagSlug: make(map[string]string),
		TagSlug2WordpressPostId: make(map[string]string),

		WordpressPostId2TagSlugs: make(map[int][]string),
		TagSlugs2WordpressPostId: make(map[string][]int),
	}

	var count int

	// Populate the authorMap.

	byteValue, err := getByteValue(config.inputs.ocaUsers)
	if err != nil {
		return nil, err
	}

	var authors []ocaAuthor

	err = json.Unmarshal(byteValue, &authors)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(authors); i++ {
		mappings.authorMap[authors[i].UserLogin] = true

		// Populate the Author ID to Author Name map
		mappings.AuthorName2AuthorId[authors[i].Name] = authors[i].Id
		mappings.AuthorId2AuthorName[authors[i].Id] = authors[i].Name

	}

	// Populate the Author Name to Sanity Ref Map.

	byteValue, err = getByteValue(config.inputs.sanityAuthorIds)
	if err != nil {
		return nil, err
	}

	r := []struct {
		AuthorName string `json:"name"`
		Id         string `json:"_id"`
	}{}

	err = json.Unmarshal(byteValue, &r)
	if err != nil {
		return nil, err
	}

	for i, v := range r {
		// If a name already exists in the dataset for OCA in our current
		// database, skip the mapping.

		mappings.SanityAuthorRef2AuthorName[v.Id] = v.AuthorName
		// fmt.Printf("Mapped %s to %s\n", v.Id, v.AuthorName)
		mappings.AuthorName2SanityAuthorRef[v.AuthorName] = v.Id
		// fmt.Printf("Mapped %s to %s\n", v.AuthorName, v.Id)
		count = i
	}

	fmt.Printf("Mapped %d authors to Sanity IDs\n", count)

	count = 0

	// Populate the Category ID to Category Slug maps.

	byteValue, err = getByteValue(config.inputs.ocaCategorySlugAndId)
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

	PrintJSON(mappings.CategorySlug2CategoryId)
	PrintJSON(mappings.CategoryId2CategorySlug)

	// Populate the Wordpress ID to Wordpress Category maps.

	byteValue, err = getByteValue(config.inputs.ocaPostIdAndCategoryId)
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

	for i, v := range d {
		mappings.WordpressId2WordpressCategoryId[v.PostId] = v.CategoryId

		_, ok := mappings.WordpressCategoryId2WordpressId[v.CategoryId]

		if !ok {
			mappings.WordpressCategoryId2WordpressId[v.CategoryId] = []int{v.PostId}
		} else {
			mappings.WordpressCategoryId2WordpressId[v.CategoryId] = append(mappings.WordpressCategoryId2WordpressId[v.CategoryId], v.PostId)
		}

		count = i
	}

	fmt.Printf("Mapped %d posts to categories\n", count)

	count = 0

	// Add Tags to Wordpress Post IDs.

	byteValue, err = getByteValue(config.inputs.ocaPostIdAndTagSlug)
	if err != nil {
		return nil, err
	}

	t := []struct {
		PostId  int    `json:"post_id"`
		TagSlug string `json:"tag_slug"`
	}{}

	err = json.Unmarshal(byteValue, &t)
	if err != nil {
		return nil, err
	}

	for i, v := range t {
		// If the entry doesn't exist, make a new array.
		if _, ok := mappings.WordpressPostId2TagSlugs[v.PostId]; !ok {
			mappings.WordpressPostId2TagSlugs[v.PostId] = make([]string, 0)
		}

		// Add the entry to the map.
		mappings.WordpressPostId2TagSlugs[v.PostId] = append(mappings.WordpressPostId2TagSlugs[v.PostId], v.TagSlug)

		count = i
	}

	fmt.Printf("Mapped %d tags to posts\n", count)

	count = 0

	// Populate Sanity Media name and Refs.

	byteValue, err = getByteValue(config.inputs.sanityMediaNameAndRefs)
	if err != nil {
		return nil, err
	}

	p := []struct {
		File    string `json:"file"`
		AssetId string `json:"assetId"`
		RefId   string `json:"refId"`
	}{}

	err = json.Unmarshal(byteValue, &p)
	if err != nil {
		return nil, err
	}

	for _, v := range p {
		mappings.SanityImageFilename2SanityImageId[v.File] = v.RefId
		mappings.SanityImageId2SanityImageFilename[v.RefId] = v.File
	}

	// Populate Sanity Tag Slugs and Refs.

	byteValue, err = getByteValue(config.inputs.sanityTagSlugAndRef)
	if err != nil {
		return nil, err
	}

	n := []struct {
		RefId string `json:"refId"`
		Slug  string `json:"slug"`
	}{}

	err = json.Unmarshal(byteValue, &n)
	if err != nil {
		return nil, err
	}

	for i, v := range n {
		mappings.TagSlug2SanityTagId[v.Slug] = v.RefId
		mappings.SanityTagId2TagSlug[v.RefId] = v.Slug
		count = i
	}

	fmt.Printf("Mapped %d unique tags to Sanity IDs\n", count)

	count = 0

	byteValue, err = getByteValue(config.inputs.sanityCategorySlugsAndId)
	if err != nil {
		return nil, err
	}

	m := []struct {
		RefId string `json:"refId"`
		Slug  string `json:"slug"`
	}{}

	err = json.Unmarshal(byteValue, &m)
	if err != nil {
		return nil, err
	}

	for i, v := range m {
		mappings.CategorySlug2SanityCategoryId[v.Slug] = v.RefId
		mappings.SanityCategoryId2CategorySlug[v.RefId] = v.Slug
		count = i
	}

	fmt.Printf("Mapped %d unique categories to Sanity IDs\n", count)

	count = 0
	return mappings, nil
}

// Data translations

// Document represents data that all Sanity documents must have.
type document struct {
	Type string `json:"_type"`
	Id   string `json:"_id,omitempty"`
	Key  string `json:"_key,omitempty"`
	Ref  string `json:"_ref,omitempty"`
}

type slug struct {
	Type    string `json:"_type"`
	Current string `json:"current"`
}

func newSlug(s string) slug {
	return slug{
		Type:    "slug",
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
