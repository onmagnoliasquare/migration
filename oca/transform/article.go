package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

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

func readBackContent(config config) ([]contentBlock, error) {
	content := []contentBlock{}

	byteValue, err := getByteValue(config.outputs.transformedBlockContent)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(byteValue, &content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func newArticle(a ocaArticle, m mappings, config config) (*article, error) {
	// To build a Sanity article, tags, content, a date, category, and an author
	// are needed.

	// The Sanity dataset uses references for the tags, category, and author.
	// These need to be mapped to their corresponding Sanity IDs, based on the
	// Wordpress name.

	// For content, references to any media must also be accounted for by their
	// Sanity ID. The content must be transformed from HTML into PortableText
	// format.

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

	// Read back the data into content.
	content, err := readBackContent(config)
	if err != nil {
		return nil, err
	}

	// Create the publish date.
	publishDate, err := time.Parse("2006-01-02 03:04:05", a.PostModifiedGMT)
	if err != nil {
		panic(err)
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

	oldCategory := m.CategoryId2CategorySlug[m.WordpressId2WordpressCategory[a.Id]]

	oldCatRefTag := newRefTag("")

	// This can become an if statement that checks if an old category is not in
	// the new categories, but this will suffice for now and is more imperative.
	switch oldCategory {
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

	// From the old category slug, retrieve the new Sanity category's ID.
	category := refCategory{
		Ref: m.CategorySlug2SanityCategoryId[m.CategoryId2CategorySlug[a.Id]],
	}

	uid := uuid.New()

	sanityArticle := &article{
		document: document{
			Type: "article",
			Id:   ocaUUID(uid),
		},

		// RFC3339Nano is YYYY-MM-DDTHH:MM:SSZ
		CreatedAt: publishDate.Format(time.RFC3339Nano),

		// DateOnly format is YYYY-MM-DD.
		Date: publishDate.Format(time.DateOnly),

		Authors:  authors,
		Category: category,
		Tags:     tags,

		Content: content,

		UseCustomCss: false,
	}

	return sanityArticle, nil
}

type contentBlock struct {
	document
	MarkDefs []string                `json:"markDefs"`
	Children []portableTextChildNode `json:"children"`
	Style    string                  `json:"style"`
}

type portableTextChildNode struct {
	document
	Marks    []string                `json:"marks"`
	Text     string                  `json:"text"`
	Children []portableTextChildNode `json:"children"`
}
