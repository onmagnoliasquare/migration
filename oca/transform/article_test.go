package main

import (
	"testing"
)

const HTML_STRING = `<html><body><h1>Title</h1></body></html>`

var c = config{
	js: js{
		inputHtmlPath:          "./js/index.html",
		indexJsPath:            "./js/index.js",
		outputHtmlPath:         "./js/output/output.html",
		transformedBlockOutput: "./js/output/transformed_block_output.json",
	},
}

func TestWriteToFile(t *testing.T) {
	c := config{
		js: js{
			inputHtmlPath: "./tests/js/input.html",
		},
	}

	got := writeToFile(HTML_STRING, c.js.inputHtmlPath)

	if got != nil {
		t.Errorf("got: %s", got)
	}
}

func TestExecNodeScript(t *testing.T) {
	t.Run("Can run real script", func(t *testing.T) {
		_, err := execNodeScript(c.js.indexJsPath)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadBackContent(t *testing.T) {
	want := []contentBlock{
		{
			document: document{Type: "block"},
			MarkDefs: []string{},
			Style:    "normal",
			Children: []contentBlock{
				{
					document: document{Type: "span"},
					Marks:    []string{},
					Text:     "Hello",
				},
			},
		},
		{
			document: document{Type: "block"},
			Children: []contentBlock{
				{
					document: document{Type: "span"},
					Marks:    []string{},
					Text:     "What is your name?",
				},
			},
			MarkDefs: []string{},
			Style:    "normal",
		},
	}

	c := config{
		js: js{
			transformedBlockOutput: "./tests/js/transformed_block_output.json",
		},
	}

	got, err := readBackContent(c.js.transformedBlockOutput)
	if err != nil {
		t.Errorf("got: %v,\n error: %q", got, err)
	}

	// fmt.Printf("%#v", want)

	// Loop through all fields and check equality.
	for i := range want {
		if got[i].document.Type != want[i].document.Type {
			t.Errorf("got: %s, want: %s", got[i].document.Type, want[i].document.Type)
		}
		if len(got[i].MarkDefs) != len(want[i].MarkDefs) {
			t.Errorf("got: %d, want: %d", len(got[i].MarkDefs), len(want[i].MarkDefs))
		}
		for j := range got[i].MarkDefs {
			if got[i].MarkDefs[j] != want[i].MarkDefs[j] {
				t.Errorf("got: %s, want: %s", got[i].MarkDefs[j], want[i].MarkDefs[j])
			}
		}
		if len(got[i].Children) != len(want[i].Children) {
			t.Errorf("got: %d, want: %d", len(got[i].Children), len(want[i].Children))
		}
		for j := range got[i].Children {
			if got[i].Children[j].document.Type != want[i].Children[j].document.Type {
				t.Errorf("got: %s, want: %s", got[i].Children[j].document.Type, want[i].Children[j].document.Type)
			}
			if len(got[i].Children[j].Marks) != len(want[i].Children[j].Marks) {
				t.Errorf("got: %d, want: %d", len(got[i].Children[j].Marks), len(want[i].Children[j].Marks))
			}
			for k := range got[i].Children[j].Marks {
				if got[i].Children[j].Marks[k] != want[i].Children[j].Marks[k] {
					t.Errorf("got: %s, want: %s", got[i].Children[j].Marks[k], want[i].Children[j].Marks[k])
				}
			}
			if got[i].Children[j].Text != want[i].Children[j].Text {
				t.Errorf("got: %s, want: %s", got[i].Children[j].Text, want[i].Children[j].Text)
			}
		}
		if got[i].Style != want[i].Style {
			t.Errorf("got: %s, want: %s", got[i].Style, want[i].Style)
		}
	}
}

func TestExtractImageFileName(t *testing.T) {
	var pathTests = map[string]struct {
		in  string
		out string
	}{
		"i2.wp.com": {
			"https://i2.wp.com/oncenturyavenue.org/wp-content/uploads/2020/09/微信图片_20200911213240.jpg?fit=640%2C502",
			"oca-2020-09-微信图片_20200911213240.jpg",
		},
		"oncenturyavenue.org": {
			"http://oncenturyavenue.org/wp-content/uploads/2018/02/media-20180214-3-1024x847.png",
			"oca-2018-02-media-20180214-3.png",
		},
	}
	for name, test := range pathTests {
		t.Run(name, func(t *testing.T) {
			// t.Parallel()
			got, err := extractImageFileName(test.in)
			if err != nil {
				t.Fatal(err)
			}

			want := test.out

			if got != want {
				t.Fatalf("extractImageFilePath(%q) returned %q; expected %q", test.in, got, want)
			}

		})
	}
}

func TestRetrieveImageRef(t *testing.T) {
	m := mappings{
		SanityImageFilename2SanityImageId: map[string]string{
			"abc.png": "id-xyz",
		},
	}

	f := retrieveImageRef(m)

	want := "id-xyz"

	got, err := f("abc.png")
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWalkContentBlocks(t *testing.T) {
	t.Run("Ingest without edit", func(t *testing.T) {
		want := "test_ref"

		content := []contentBlock{
			{
				document: document{
					Type: "image",
				},
				Asset: &reference{Ref: want, Type: "reference"},
			},
		}

		edits := []func(s string) (string, error){
			func(s string) (string, error) { return s, nil },
		}

		c, err := walkContentBlocks("image", content, edits)
		if err != nil {
			t.Fatal(err)
		}

		got := c[0].Asset.Ref

		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("Edits correctly", func(t *testing.T) {
		want := "edited-correctly"

		content := []contentBlock{
			{
				document: document{
					Type: "image",
				},
				Asset: &reference{Ref: "ref", Type: "reference"},
			},
		}

		edits := []func(s string) (string, error){
			func(s string) (string, error) {
				return want, nil
			},
		}

		content, err := walkContentBlocks("image", content, edits)
		if err != nil {
			t.Fatal(err)
		}

		if content[0].Asset.Ref != want {
			t.Errorf("got %s, want %s", content[0].Asset.Ref, want)
		}
	})
}

func TestNewArticle(t *testing.T) {
	m := mappings{
		AuthorId2AuthorName:               map[int]string{13: "Tom Sawyer"},
		AuthorName2SanityAuthorRef:        map[string]string{"Tom Sawyer": "sanity-author-ref-id"},
		TagSlug2SanityTagId:               map[string]string{"fake-tag": "sanity-tag-id-abcdefg", "on-century-avenue": "sanity-tag-oca", "random-tag": "sanity-random-tag"},
		WordpressId2WordpressCategoryId:   map[int]int{154: 9},
		CategoryId2CategorySlug:           map[int]string{9: "news"},
		SanityCategories:                  map[string]bool{"news": true},
		WordpressPostId2TagSlugs:          map[int][]string{154: {"random-tag"}},
		CategorySlug2SanityCategoryId:     map[string]string{"news": "sanity-category-id-abcdefg"},
		SanityImageFilename2SanityImageId: map[string]string{"oca-2014-02-wskiddlydoo.jpg": "image-abcdefg-543x123-jpg"},
	}

	a := ocaArticle{
		Id:          154,
		PostAuthor:  13,
		PostName:    "the-grey-horse-nebraska",
		PostTitle:   `The Grey Horse: \"Nebraska\"`,
		PostDateGMT: "2014-02-01 16:49:40",
		PostExcerpt: "",
		PostContent: `"<p dir=\"ltr\">Despite being raised and educated in the United States, I doubt that I could place it on a map. I know it’s one of those ambiguous square states that reside in the middle. But, where? Well, that would only be a guess. In fact, in light of GPS guided thinking, I’m starting to wonder if the place even exists, since I have never met anyone from this so-called “Nebraska.” But perhaps that’s taking it a step too far. Even though I don’t know Nebraska, I have this innate connotation, like many others, that it is this boxy, flat terrain -- tarred in this sort of boredom that makes no one talk about it. It is for this reason that I was utterly surprised when I started seeing the nominees of this year’s upcoming Academy Awards plastered with this state -- Nebraska.</p>\r\n<p dir=\"ltr\">Going into the movie theater, I didn’t know what to expect, but I was immediately oppressed by the smell of hard candies and leathered skin. I wondered if I had made a mistake coming to a movie that attracted a demographic where the youngest person in the theater (besides me) was starting to develop their training cataracts. But I sat down anyways, clutching my American-oversized ICEE, committed to the ride. <img src="https://wordpress.com/oca/2014/02/wskiddlydoo-543x123.jpg" /></p>\r\n<p dir=\"ltr\">And it was a ride that was freshly unique. It wasn’t a rehash of the same Hollywood plotline told with a different cast of characters and a new director. It wasn’t a sequel, trilogy, or film series of similar ideas. It was a piece that stood entirely on its own, with its own distinctive voice. Cast in stark black-and-white cinematography, the movie starts off with Woody Grant (Bruce Durn), an older man with large wire-framed glasses and sparse, white cotton candy hair, that has made wandering his habit. After he gets picked up off the side of the road by a police officer, Woody’s son, David Grant (Will Forte), discovers that his aging father believes that he has won a million dollars from an ambiguous weekly advert and has to go to Nebraska to pick up his winnings. Wanting to break his fathers fantasy, once and for all, David agrees to take his father on a sobering road trip to Nebraska. The trip traces the tale of Woody’s life, as miles on the road unearth the ever fading past. It is in this endearing tale, with a  colorful character set that needs no color, that a father’s forgotten dreams, goals, and aspirations are revived and revisited.</p>\r\nBut what were Nebraska’s aspirations  for the Academy Awards?  It was clear from the beginning that if the Academy Awards were a horse race, “Nebraska” wouldn’t be the horse to vote on.<img src="https://googleusercontent.com/2018/10/asdlkjaskljdklajds-243x135.png" /> It was the significantly older grey horse with the slick silver mane and stiffened joints. It was the long shot. But you don’t watch “Nebraska” because it was destined to win. In fact, it seems that “Nebraska” has all but been forgotten now that the race is over. You watch “Nebraska” because it made it to the race, despite all odds. You watch “Nebraska” because it is an endearing two hours you will not regret.\r\n\r\n<hr size=\"2\" />\r\n<small>This article was written by <em>Tyler Rhorick</em>. Send an email to <strong>oncenturyavenue@gmail.com</strong> to get in touch.\r\n<strong>Photo Credit:</strong> Paramount Vantage</small>"`,
	}

	article, err := newArticle(a, m, c)
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("%+v", article)
	// PrintJSON(article)

	// Check if the author ref was successfully referred to.
	authorRef := article.Authors[0].Ref
	if authorRef != "sanity-author-ref-id" {
		t.Errorf("got %s, want %s", authorRef, "sanity-author-ref-id")
	}

	// Check if the OCA tag is present.
	ocaTag := article.Tags[0].Ref
	if ocaTag != "sanity-tag-oca" {
		t.Errorf("got %s, want %s", ocaTag, "sanity-tag-oca")
	}

	// Check if the date is correct.
	articleDate := article.Date
	if articleDate != "2014-02-01" {
		t.Errorf("got %s, want %s", articleDate, "2014-02-01")
	}

	// Check if category is correct.
	articleCategory := article.Category
	if articleCategory.Ref != "sanity-category-id-abcdefg" {
		t.Errorf("got %s, want %s", articleCategory.Ref, "sanity-category-id-abcdefg")
	}

	// Check if slug is correct.
	articleSlug := article.Slug.Current
	if articleSlug != "the-grey-horse-nebraska" {
		t.Errorf("got %s, want %s", articleSlug, "the-grey-horse-nebraska")
	}

	// Check if article title is correct.
	articleTitle := article.Title
	if articleTitle != `The Grey Horse: "Nebraska"` {
		t.Errorf("got %s, want %s", articleTitle, `The Grey Horse: "Nebraska"`)
	}

	// Check if there is an image and its correct. The input content will only
	// have one valid image type, so the checks can be exact values.
	for _, v := range article.Content {
		if v.Type == "image" {
			if v.Alt != "replace this alt text" {
				// If this happens, might want to check if the index.js file creates
				// the exact string "replace this alt text" when there is no alt
				// text available.
				t.Errorf("got %s, want %s", v.Alt, "replace this alt text")
			}

			if v.Asset.Type != "reference" {
				t.Errorf("got %s, want %s", v.Asset.Type, "reference")
			}

			if v.Asset.Ref != "image-abcdefg-543x123-jpg" {
				t.Errorf("got %s, want %s", v.Asset.Ref, "image-abcdefg-543x123-jpg")
			}
		}
	}
}
