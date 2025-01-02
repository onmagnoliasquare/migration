package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// article mirrors the Sanity schema article document.
type article struct {
	document
	Slug         slug           `json:"slug"`
	Title        string         `json:"title"`
	Subtitle     string         `json:"subtitle"`
	Abstract     string         `json:"abstract"`
	Authors      []refAuthor    `json:"authors"`
	Date         string         `json:"date"`
	CreatedAt    string         `json:"_createdAt"`
	UpdatedAt    string         `json:"_updatedAt"`
	Content      []contentBlock `json:"content"`
	Tags         []refTag       `json:"tags"`
	Category     refCategory    `json:"category"`
	UseCustomCss bool           `json:"useCustomCss"`
}

// newArticle creates a new article struct that conforms to our Sanity schema.
// To build a Sanity article, tags, content, a date, category, and an author
// are needed.
//
// The Sanity dataset uses references for the tags, category, and author.
// These need to be mapped to their corresponding Sanity IDs, based on the
// Wordpress name.
//
// For content, references to any media must also be accounted for by their
// Sanity ID. The content must be transformed from HTML into PortableText
// format. This happens in the js directory.
//
// In the future, I may write a parsing library based on the @sanity/
// sanity-blocks parser written in TypeScript. - Neo
func newArticle(a ocaArticle, m mappings, config config) (*article, error) {

	// Write the post_content to an HTML file.
	err := writeToFile(a.PostContent, config)
	if err != nil {
		return nil, err
	}

	// Execute the node script to transform HTML into PortableText blocks.
	_, err = execNodeScript(config)
	if err != nil {
		return nil, err
	}

	// Read back the JSON data into content.
	content, err := readBackContent(config)
	if err != nil {
		return nil, err
	}

	// Replace any image _ref in content with the corresponding _ref IDs from
	// Sanity.

	edits := []func(s string) (string, error){
		extractImageFileName,
		retrieveImageRef(m),
	}

	// Walk the content block array and find any element that matches the
	// "image" _type. When found, apply the edit functions to the image type's
	// _ref element.
	content, err = walkContentBlocks("image", content, edits)
	if err != nil {
		return nil, err
	}

	// Create the publish date.
	publishDate, err := time.Parse("2006-01-02 03:04:05", a.PostModifiedGMT)
	if err != nil {
		return nil, err
	}

	// There is no need to populate the array here with multiple authors,
	// because the Wordpress articles only have one author each.
	authors := []refAuthor{
		{
			document: document{
				Type: "author",
			},
			Ref: m.Id2AuthorRef[a.PostAuthor],
		},
	}

	tags := []refTag{}
	tags = append(tags, newRefTag(m.TagSlug2SanityTagId["on-century-avenue"]))

	// If a category no longer exists, add the corresponding existing tag to
	// the new Sanity Article.

	oldCategorySlug := m.CategoryId2CategorySlug[m.WordpressId2WordpressCategoryId[a.Id]]

	if _, ok := m.sanityCategories[oldCategorySlug]; !ok {

		oldCatRefTag := newRefTag("")

		switch oldCategorySlug {

		// This can become an if statement that checks if an old category is
		// not in the new categories, but this will suffice for now and is more
		// imperative.

		case "%e4%b8%ad%e6%96%87":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["中文"]
		case "student-government":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["student-government"]
		case "campus-life":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["campus-life"]
		case "events":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["events"]
		case "food-and-nightlife":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["food-and-nightlife"]
		case "fashion":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["fashion"]
		case "lifestyle":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["lifestyle"]
		case "off-campus":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["off-campus"]
		case "how-to-get-an-a":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["how-to-get-an-a"]
		case "multilingual":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["multilingual"]
		case "business-and-economics":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["business-and-economics"]
		case "chineseglobal":
			oldCatRefTag.Id = m.TagSlug2SanityTagId["china-global"]
		}

		tags = append(tags, oldCatRefTag)
	}

	// Then according to the old tags of the article, append those too.

	tagSlugs := m.WordpressPostId2TagSlugs[a.Id]
	for _, v := range tagSlugs {
		tags = append(tags, newRefTag(m.TagSlug2SanityTagId[v]))
	}

	category := refCategory{
		// From the old category slug, retrieve the new Sanity category's ID.
		Ref: m.CategorySlug2SanityCategoryId[m.CategoryId2CategorySlug[m.WordpressId2WordpressCategoryId[a.Id]]],
	}

	uid := uuid.New()

	sanityArticle := &article{
		document: document{
			Type: "article",
			Id:   ocaUUID(uid),
		},

		// RFC3339Nano is YYYY-MM-DDTHH:MM:SSZ
		CreatedAt: publishDate.Format(time.RFC3339Nano),

		Title: a.PostTitle,
		Slug:  newSlug(a.PostName),

		// DateOnly format is YYYY-MM-DD.
		Date: publishDate.Format(time.DateOnly),

		Authors:      authors,
		Category:     category,
		Tags:         tags,
		Content:      content,
		UseCustomCss: false,
	}

	// Some posts have a populated "post_excerpt" field. If its not blank,
	// use it as the subtitle.
	if a.PostExcerpt != "" {
		sanityArticle.Subtitle = a.PostExcerpt
	}

	return sanityArticle, nil
}

// writeToFile writes an HTML string to an HTML file at the path specified
// by config.
func writeToFile(html string, config config) error {
	err := os.WriteFile(config.js.inputHtmlPath, []byte(html), 0644)
	if err != nil {
		return err
	}
	return nil
}

// execNodeScript executes a Node.js script at the path specified by the
// indexJsPath field of the js struct. The function returns the output of the
// script as a byte array, or an error if the command fails.
func execNodeScript(config config) ([]byte, error) {
	cmd := exec.Command(NODE_PATH, config.js.indexJsPath)

	d, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute node command: %s", err)
	}

	return d, nil
}

// readBackContent unmarshals the JSON data from config's
// transformedBlockContent into an array of contentBlocks.
func readBackContent(config config) ([]contentBlock, error) {
	content := []contentBlock{}

	byteValue, err := getByteValue(config.js.transformedBlockContent)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(byteValue, &content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// extractImageFilePath parses a URL and extracts the YYYY/MM/{file}.{ext}
// path. Then, it changes the forward-slashes into hyphens and  removes the
// image dimensions from the file URL.
func extractImageFileName(s string) (string, error) {
	var sb strings.Builder

	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	elms := strings.Split(u.Path, "/")

	// Get the last three elements of the array and join them with -. The last
	// three elements are the file name and the date in YYYY-MM format.
	_, err = sb.WriteString(strings.Join(elms[len(elms)-3:], "-"))
	if err != nil {
		return "", err
	}

	// Remove the image dimension from the URL.

	filename := strings.Join([]string{"oca-", sb.String()}, "")

	// Find the last dot of the filename, i.e. the dot before the extension.
	dotIndex := strings.LastIndex(filename, ".")

	// If the dot has no index, return the filename as is.
	if dotIndex == -1 {
		return filename, nil
	}

	// Take a slice of the filename from the beginning UP TO the dot index, and
	// select the final hyphen which indicates the dimension information.
	dimensionIndex := strings.LastIndex(filename[:dotIndex], "-")

	// If no dimensions are found, return string as is.
	if dimensionIndex == -1 || !strings.Contains(filename[dimensionIndex:], "x") {
		return filename, nil
	}

	return filename[:dimensionIndex] + filename[dotIndex:], nil
}

// retrieveImageRef uses a file name to get the corresponding mapping of the
// image ID that exists on Sanity Content Lake. File names are prefixed with
// "oca-". Article image refs on Sanity take the form of:
// "image-{Asset ID}-{width}x{height}-{ext}".
func retrieveImageRef(m mappings) func(s string) (string, error) {
	return func(s string) (string, error) {
		// Use filename as key and retrieve the value from the filename to ref
		// ID map. These IDs are from Sanity.
		ref, ok := m.SanityImageFilename2SanityImageId[s]
		if !ok {
			return "", fmt.Errorf("image filename not in Sanity mapping: %s", s)
		}

		return ref, nil
	}
}

type contentBlock struct {
	document
	MarkDefs []string       `json:"markDefs,omitempty"`
	Children []contentBlock `json:"children"`
	Style    string         `json:"style,omitempty"`
	Marks    []string       `json:"marks,omitempty"`

	// Span node type
	Text string `json:"text,omitempty"`

	// Image block type
	Alt   string    `json:"alt,omitempty"`
	Asset reference `json:"asset,omitempty"`
}

// walkContentBlocks walks through each child of a contentBlock array. It finds
// a given "Type" and changes the value according to the edit functions.
func walkContentBlocks(t string, c []contentBlock, edits []func(s string) (string, error)) ([]contentBlock, error) {
	if len(c) < 1 {
		return c, nil
	}

	for i := range c {
		// If there are children, recursively walk through them
		if len(c[i].Children) > 0 {
			var err error
			c[i].Children, err = walkContentBlocks(t, c[i].Children, edits)
			if err != nil {
				return nil, err
			}
		}

		// If the block type matches, apply edits to its asset reference
		if c[i].document.Type == t {
			for _, edit := range edits {
				var err error
				c[i].Asset.Ref, err = edit(c[i].Asset.Ref)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return c, nil
}

type reference struct {
	Ref  string `json:"_ref"`
	Type string `json:"_type"`
}

// refAuthor mirrors the author reference field of a Sanity document.
type refAuthor struct {
	document

	// Ref is the `_id` of the Member document.
	// See: https://www.sanity.io/docs/reference-type#e97572ca6050
	Ref string `json:"_ref"`
}

// refCategory mirrors the category reference field of a Sanity document.
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

// refTag mirrors the tag reference field of a Sanity document.
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
