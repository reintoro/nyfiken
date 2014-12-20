package strip

import (
	"bytes"
	"strings"
	"testing"

	"github.com/karlek/nyfiken/settings"
	"golang.org/x/net/html"
)

// NOTE: Make sure to use t.Fatal instead of t.Error if the test cannot continue
// after an error is encountered. It is conventional to use got, want and golden
// for test cases; of cause you can change to whatever feels right for you. It's
// great that you've implemented test cases! Updating and testing strip.HTML was
// much easier with a test case to rely on. The test case was using an older
// version of the strip API but updating it was quite trivial. To prevent a skew
// from happening in the future start using Continuous Integration and maybe
// even git commit hooks which runs go test, golint, etc. Have a look at
// goclean.sh [1] and an example repo using it [2].
//
// [1]: https://gist.github.com/mewmew/379014c9a2e6885e238d
// [2]: https://github.com/mewrev/pe/blob/master/.travis.yml
//
//    [~/go/src]$ ack -l golden
//    crypto/sha512/sha512_test.go
//    ...
//    image/draw/draw_test.go
//    ...
//    time/time_test.go

func TestNumber(t *testing.T) {
	node1, err := html.Parse(strings.NewReader("<html><head><title>Number test 12345</title></head><body><b>I am a number 2!</b></body></html>"))
	if err != nil {
		t.Fatal("error:", err)
	}

	var golden = []struct {
		input *html.Node
		want  string
	}{
		{node1, "<html><head><title>Number test </title></head><body><b>I am a number !</b></body></html>"},
	}

	buf := new(bytes.Buffer)
	for _, g := range golden {
		Numbers(g.input)
		err = html.Render(buf, g.input)
		if err != nil {
			t.Error("error:", err)
			continue
		}
		got := buf.String()
		if got != g.want {
			t.Errorf("output `%v` != expected `%v`", got, g.want)
		}
	}
}

func TestAttrs(t *testing.T) {
	node1, err := html.Parse(strings.NewReader(`<html><head><title>Attr test</title></head><body><b style="color: #f00;">I am red!</b></body></html>`))
	if err != nil {
		t.Errorf("error: %s", err)
	}

	var golden = []struct {
		input *html.Node
		want  string
	}{
		{node1, `<html><head><title>Attr test</title></head><body><b>I am red!</b></body></html>`},
	}

	buf := new(bytes.Buffer)
	for _, g := range golden {
		Attrs(g.input)
		err = html.Render(buf, g.input)
		if err != nil {
			t.Error("error:", err)
			continue
		}
		got := buf.String()
		if got != g.want {
			t.Errorf("output `%v` != expected `%v`", got, g.want)
		}
	}
}

// NOTE: The tests for strip.HTML is currently failing. The reason is that our
// test case uses html.Render which always includes <html> and <head> tags. Any
// ideas on a clean way to test this function without making the test case
// implement too much custom code (which may be inaccurate)?
//
// Example output:
//    --- FAIL: TestHTML (0.00s)
//    	strip_test.go:111: output `<html><head></head><body>HTML test
//    		I am red!
//    		</body></html>` != expected `HTML test
//    		I am red!
//    		`

func TestHTML(t *testing.T) {
	node1, err := html.Parse(strings.NewReader(`<html><head><title>HTML test</title></head><body><b style="color: #f00;">I am red!</b></body></html>`))
	if err != nil {
		t.Errorf("error: %s", err)
	}

	var golden = []struct {
		input *html.Node
		want  string
	}{
		{node1, `HTML test` + settings.Newline + `I am red!` + settings.Newline},
	}

	buf := new(bytes.Buffer)
	for _, g := range golden {
		HTML(g.input)
		err = html.Render(buf, g.input)
		if err != nil {
			t.Error("error:", err)
			continue
		}
		got := buf.String()
		if got != g.want {
			t.Errorf("output `%v` != expected `%v`", got, g.want)
		}
	}
}
