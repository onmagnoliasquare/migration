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
			Children: []portableTextChildNode{
				{
					document: document{Type: "span"},
					Marks:    []string{},
					Text:     "Hello",
				},
			},
		},
		{
			document: document{Type: "block"},
			Children: []portableTextChildNode{
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
