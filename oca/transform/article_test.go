package main

import (
	"fmt"
	"testing"
)

const HTML_STRING = `<html><body><h1>Title</h1></body></html>`

func TestWriteToFile(t *testing.T) {
	c := config{
		js: js{
			inputHtmlPath: "./tests/js/input.html",
		},
	}

	got := writeToFile(HTML_STRING, c)

	if got != nil {
		t.Errorf("got: %s", got)
	}
}

func TestExecNodeScript(t *testing.T) {
	c := config{
		js: js{
			indexJsPath: "./tests/js/index.js",
		},
	}

	d, got := execNodeScript(c)

	if got != nil {
		t.Errorf("got: %s", got)
	}

	fmt.Println(string(d))
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
			transformedBlockContent: "./tests/js/transformed_block_output.json",
		},
	}

	got, err := readBackContent(c)
	if err != nil {
		t.Errorf("got: %s,\n error: %s", got, err)
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
				Asset: reference{Ref: want, Type: "reference"},
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
				Asset: reference{Ref: "ref", Type: "reference"},
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
	a := ocaArticle{}
	m := mappings{}
	c := config{}

	article, err := newArticle(a, m, c)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%#v", article)
}
