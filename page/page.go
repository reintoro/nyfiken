// Package page contains functions which checks if a page has been updated.
package page

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"code.google.com/p/cascadia"
	"code.google.com/p/mahonia"
	"github.com/karlek/nyfiken/distance"
	"github.com/karlek/nyfiken/filename"
	"github.com/karlek/nyfiken/mail"
	"github.com/karlek/nyfiken/settings"
	"github.com/karlek/nyfiken/strip"
	"github.com/mewkiz/pkg/errutil"
	"github.com/mewkiz/pkg/htmlutil"
	"golang.org/x/net/html"
)

// Page is a site which is checked for changes. It has specialized settings to
// eliminate false-positives.
type Page struct {
	ReqUrl   *url.URL
	Settings settings.Page
}

// NOTE: As filename.Encode is always used in combination with UrlAsFilename,
// move the Encode function call into the defintion of UrlAsFilename. Make sure
// to update the call locations in both page/page.go and
// cmd/nyfikend/nyfikend.go.

// NOTE: golint would complain about Url and recommend using URL instead as it
// is an acronym. If you agree with this advice is entirely up to you.

// NOTE: Make sure that UrlAsFilename always returns a unique representation.
// E.g. are these URLs encoded differently?
//    foo.in/fobar
//    foo.info/bar

// UrlAsFilename returns a unique representation of the page's URL with illegal
// filesystem characters encoded.
func (p *Page) UrlAsFilename() string {
	return p.ReqUrl.Host + p.ReqUrl.Path + p.ReqUrl.RawQuery
}

// NOTE: Avoid concurrency in the exposed API [1]. As some of the packages of
// nyfikend are quite tightly integrated this may not be an issue, where the
// concurrent version of Check works as a helper function. For standalone
// packages concurrency should be avoided in the API, as clients of the API may
// want to use serial execution and executing a non-concurrent function
// concurrently is easy.
//
// [1] http://talks.golang.org/2013/bestpractices.slide#25

// Check downloads and makes a specialized comparison with a previous check
// saved on disk to determine if the page has been updated. Check takes
// an error channel to concurrently handle errors.
func (p *Page) Check(ch chan<- error) {
	ch <- p.check()
}

// NOTE: The check function implements a lot of functionality and is massive
// (more than 160 lines). Consider factoring out some functionality to dedicated
// functions; for instance the sending of email notifications. Generally a
// function should fit on a screen or two; anything more will make it difficult
// to keep it all in your head.

// check is an non-exported function for better error handling.
func (p *Page) check() (err error) {
	if settings.Verbose {
		fmt.Println("[/] Downloading:", p.ReqUrl.String())
	}

	// NOTE: Use embedding to propagate methods of a given struct member, not to
	// save key strokes. As *html.Node and error are embedded members of r the
	// struct will have the Error, AppendChild, InsertBefore and RemoveChild
	// methods.
	//
	//    var r struct { node *html.Node; err error }

	// Retrieve result from download or return timeout error.
	var r struct {
		*html.Node
		error
	}
	// NOTE: Ideomatic use of select and time.After for timeouts, nice :)
	select {
	case r = <-errWrapDownload(p):
		if r.error != nil {
			return errutil.Err(r.error)
		}
	case <-time.After(settings.TimeoutDuration):
		return errutil.NewNoPosf("timeout: %s", p.ReqUrl.String())
	}

	// Extract selection from downloaded source.
	selection, err := p.makeSelection(r.Node)
	if err != nil {
		return errutil.Err(err)
	}

	// NOTE: Rename linuxPath to pagePath or something, as nyfiken may be running
	// on other operating systems.

	// Filename is the URL encoded and the protocol is stripped.
	linuxPath, err := filename.Encode(p.UrlAsFilename())
	if err != nil {
		return errutil.Err(err)
	}

	// Debug - no selection.
	debug, err := htmlutil.RenderClean(r.Node)
	if err != nil {
		return errutil.Err(err)
	}
	// Update the debug comparison file.

	// NOTE: Consider using filepath.Join instead of string appends. The ".htm"
	// string still needs to be appended as it is part of the filename and not a
	// distinct directory path. E.g.
	//
	//    debugCachePathName := filepath.Join(settings.DebugCacheRoot, linuxPath + ".htm")
	debugCachePathName := settings.DebugCacheRoot + linuxPath + ".htm"
	err = ioutil.WriteFile(debugCachePathName, []byte(debug), settings.Global.FilePerms)
	if err != nil {
		return errutil.Err(err)
	}

	// NOTE: This check has saved me quite a few times :)

	// If the selection is empty, the CSS selection is probably wrong so we will
	// alert the user about this problem.
	if len(selection) == 0 {
		return errutil.NewNoPosf("Update was empty. URL: %s", p.ReqUrl)
	}

	cachePathName := settings.CacheRoot + linuxPath + ".htm"

	// Read in comparison.
	buf, err := ioutil.ReadFile(cachePathName)
	if err != nil {
		if !os.IsNotExist(err) {
			return errutil.Err(err)
		}

		// If the page hasn't been checked before, create a new comparison file.
		err = ioutil.WriteFile(
			cachePathName,
			[]byte(selection),
			settings.Global.FilePerms,
		)
		if err != nil {
			return errutil.Err(err)
		}

		readPathName := settings.ReadRoot + linuxPath + ".htm"
		// If the page hasn't been checked before, create a new comparison file.
		err = ioutil.WriteFile(
			readPathName,
			[]byte(selection),
			settings.Global.FilePerms,
		)
		if err != nil {
			return errutil.Err(err)
		}

		debugReadPathName := settings.DebugReadRoot + linuxPath + ".htm"

		// Update the debug prev file.
		err = ioutil.WriteFile(debugReadPathName, []byte(debug), settings.Global.FilePerms)
		if err != nil {
			return errutil.Err(err)
		}

		if settings.Verbose {
			fmt.Println("[+] New site added:", p.ReqUrl.String())
		}

		return nil
	}

	// The distance between to strings in percentage.
	dist := distance.Approx(string(buf), selection)

	// If the distance is within the threshold level, i.e if the check was a
	// match.
	if dist > p.Settings.Threshold {
		u := p.ReqUrl.String()
		settings.Updates[u] = true

		if settings.Verbose {
			fmt.Println("[!] Updated:", p.ReqUrl.String())
		}

		// If the page has a mail and all compulsory global mail settings are
		// set, send a mail to notify the user about an update.
		if p.Settings.RecvMail != "" &&
			settings.Global.SenderMail.AuthServer != "" &&
			settings.Global.SenderMail.OutServer != "" &&
			settings.Global.SenderMail.Address != "" {

			// Mail the selection without the stripping functions, since their
			// only purpose is to remove false-positives. It will make the
			// output look better.
			mailPage := Page{p.ReqUrl, p.Settings}
			mailPage.Settings.StripFuncs = nil
			mailPage.Settings.Regexp = ""
			sel, err := mailPage.makeSelection(r.Node)
			if err != nil {
				return errutil.Err(err)
			}

			err = mail.Send(p.ReqUrl, p.Settings.RecvMail, sel)
			if err != nil {
				return errutil.Err(err)
			}
			delete(settings.Updates, u)
		}
		// Save updates to file.
		err = settings.SaveUpdates()
		if err != nil {
			return errutil.Err(err)
		}

		// Update the comparison file.
		err = ioutil.WriteFile(cachePathName, []byte(selection), settings.Global.FilePerms)
		if err != nil {
			return errutil.Err(err)
		}
	} else {
		if settings.Verbose {
			fmt.Println("[-] No update:", p.ReqUrl.String())
		}
	}
	return nil
}

// NOTE: Consider adding an unexported type name for struct{*html.Node;error} as
// it is used several times, and if so make sure to update both check and
// errWrapDownload.

// An error wrapping convenience function for p.download() used because of
// timeout implementation.
// Credits to: Dave Cheney and ilyia (https://groups.google.com/forum/?fromgroups=#!topic/golang-nuts/cTrBcyjqCxg)
func errWrapDownload(p *Page) <-chan struct {
	*html.Node
	error
} {
	doc, err := p.download()
	result := make(chan struct {
		*html.Node
		error
	})
	go func() {
		result <- struct {
			*html.Node
			error
		}{doc, err}
	}()
	return result
}

// Download the page with or without user specified headers.
func (p *Page) download() (doc *html.Node, err error) {

	// Construct the request.
	req, err := http.NewRequest("GET", p.ReqUrl.String(), nil)
	if err != nil {
		return nil, errutil.Err(err)
	}

	// NOTE: Simple and clean way of adding custom HTTP headers to a request.

	// If special headers were specified, add them to the request.
	if p.Settings.Header != nil {
		for key, val := range p.Settings.Header {
			req.Header.Add(key, val)
		}
	}

	// Do request and read response.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if serr, ok := err.(*url.Error); ok {
			if serr.Err == io.EOF {
				return nil, errutil.NewNoPosf("Update was empty: %s", p.ReqUrl)
			}
		}
		return nil, errutil.Err(err)
	}
	defer resp.Body.Close()

	// If response contained a client or server error, fail with that error.
	if resp.StatusCode >= 400 {
		return nil, errutil.Newf("%s: (%d) - %s", p.ReqUrl.String(), resp.StatusCode, resp.Status)
	}

	// Read the response body to []byte.
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errutil.Err(err)
	}

	// Fix charset problems with servers that doesn't use utf-8
	charset := "utf-8"
	content := string(buf)

	types := strings.Split(resp.Header.Get("Content-Type"), ` `)
	for _, typ := range types {
		if strings.Contains(typ, "charset") {
			keyval := strings.Split(typ, `=`)
			if len(keyval) == 2 {
				charset = keyval[1]
				// NOTE: I've added a break for when the first charset is found. If
				// the Content-Type contains multiple charsets this may not be
				// wanted. What do you think is the right behaviour?
				break
			}
		}
	}
	if charset != "utf-8" {
		// NOTE: I like the fact that you handle non UTF-8 encodings! I've not
		// come across any encoding issues while running nyfiken, maybe this is
		// why.
		content = mahonia.NewDecoder(charset).ConvertString(content)
	}
	// Parse response into html.Node.
	return html.Parse(strings.NewReader(content))
}

// NOTE: Definitely break the makeSelection function into smaller functions.
// Right now you are using --- [ foo ] --- to separate the functionality, so
// split it instead.

// Select from the retrived page source the CSS selection defined in c4c.ini.
func (p *Page) makeSelection(htmlNode *html.Node) (selection string, err error) {

	// --- [ CSS selection ] --------------------------------------------------/

	// Write results into an array of nodes.
	var result []*html.Node

	// Append the whole page (htmlNode) to results if no selector where chosen.
	if p.Settings.Selection == "" {
		result = append(result, htmlNode)
	} else {

		// Make a selector from the user specified string.
		s, err := cascadia.Compile(p.Settings.Selection)
		if err != nil {
			return "", errutil.Err(err)
		}

		// Find all nodes that matches selection s.
		result = s.MatchAll(htmlNode)
	}

	// Loop through all the hits and render them to string.
	for _, hit := range result {
		s, err := htmlutil.RenderClean(hit)
		if err != nil {
			return "", errutil.Err(err)
		}
		selection += s
	}

	// --- [ /CSS selection ] -------------------------------------------------/

	// --- [ Strip funcs ] ----------------------------------------------------/

	for _, stripFunc := range p.Settings.StripFuncs {
		doc, err := html.Parse(strings.NewReader(selection))
		if err != nil {
			return "", errutil.Err(err)
		}
		// NOTE: This is one of the features I enjoy the most with nyfiken! The
		// strip functions are simply great. Consider adding a default case which
		// catches invalid or misspelled strip functions. I know that the
		// nyfiken/ini package already takes care of this, but a future version of
		// nyfiken may replace the configuration implementation and forget this
		// check. In general, use defence in depth when checking for errors. I
		// know this is a silly example but the principle is the same.
		stripFunc = strings.ToLower(stripFunc)
		switch stripFunc {
		case "numbers":
			strip.Numbers(doc)
		case "attrs":
			strip.Attrs(doc)
		case "html":
			strip.HTML(doc)
		case "scripts":
			strip.Scripts(doc)
		}

		selection, err = htmlutil.RenderClean(doc)
		if err != nil {
			return "", errutil.Err(err)
		}
	}

	// --- [ /Strip funcs ] ---------------------------------------------------/

	// --- [ Regexp ] ---------------------------------------------------------/

	if p.Settings.Regexp != "" {
		re, err := regexp.Compile(p.Settings.Regexp)
		if err != nil {
			return "", errutil.Err(err)
		}

		// -1 means to find all.
		result := re.FindAllString(selection, -1)

		selection = ""
		for _, res := range result {
			selection += res + settings.Newline
		}
	}

	// --- [ /Regexp ] --------------------------------------------------------/

	// --- [ Negexp ] ---------------------------------------------------------/

	if p.Settings.Negexp != "" {
		ne, err := regexp.Compile(p.Settings.Negexp)
		if err != nil {
			return "", errutil.Err(err)
		}

		// Remove all that matches the regular expression ne
		selection = ne.ReplaceAllString(selection, "")
	}

	// --- [ /Negexp ] --------------------------------------------------------/

	return selection, nil
}

// Check all pages immediately
func ForceUpdate(pages []*Page) (err error) {
	// A channel in which errors are sent from p.Check()
	errChan := make(chan error)

	// The number of checks currently taking place
	var numChecks int
	for _, p := range pages {
		// Start a go-routine to check if the page has been updated.
		go p.Check(errChan)
		numChecks++
	}

	// For each check that took place, listen if any check returned an error
	go func(ch chan error, nChecks int) {
		for i := 0; i < nChecks; i++ {
			if err := <-ch; err != nil {
				log.Println(errutil.Err(err))
			}
		}
	}(errChan, numChecks)

	return nil
}
