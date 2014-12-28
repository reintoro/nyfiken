package ini

import (
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/karlek/nyfiken/page"
	"github.com/karlek/nyfiken/settings"
)

// Tests ReadSettings
func TestReadSettings(t *testing.T) {
	// Expected output of ReadSettings.
	expected := settings.Prog{
		Interval:  10 * time.Minute,
		RecvMail:  "global@example.com",
		FilePerms: os.FileMode(0777),
		PortNum:   ":4113",
		Browser:   "/usr/bin/browser",

		SenderMail: struct {
			Address    string
			Password   string
			AuthServer string
			OutServer  string
		}{
			Address:    "sender@example.com",
			Password:   "123456",
			AuthServer: "auth.server.com",
			OutServer:  "out.server.com:587",
		},
	}

	err := ReadSettings("ini_test_config.ini")
	if err != nil {
		t.Fatal("ReadSettings:", err)
	}

	// NOTE: As you already pointed out the fmt solution was ugly although
	// creative. reflect.DeepEqual can be used instead.
	if !reflect.DeepEqual(settings.Global, expected) {
		t.Errorf("output %v != %v", settings.Global, expected)
	}
}

// Tests ReadPages
func TestReadPages(t *testing.T) {
	reqUrl, err := url.Parse("http://example.org")
	if err != nil {
		t.Fatal("url.Parse:", err)
	}
	anotherReqUrl, err := url.Parse("http://another.example.org")
	if err != nil {
		t.Fatal("url.Parse:", err)
	}

	expected := []*page.Page{
		{
			ReqUrl: reqUrl,
			Settings: settings.Page{
				Interval:  3 * time.Minute,
				Threshold: 0.05,
				RecvMail:  "mail@example.org",
				Selection: "html body",
				StripFuncs: []string{
					"html",
					"numbers",
				},
				Regexp: "(love)",
				Negexp: "(hate)",
				Header: map[string]string{
					"Cookie":     "IloveCookies=1;",
					"User-Agent": "I come in peace",
				},
			},
		},
		{
			ReqUrl: anotherReqUrl,
			Settings: settings.Page{
				Interval:  settings.Global.Interval,
				RecvMail:  settings.Global.RecvMail,
				Selection: "#main-content",
				// NOTE: Added since reflect.DeepEqual differentiates between nil
				// maps and empty (but initialized) maps.
				Header: map[string]string{},
			},
		},
	}

	pages, err := ReadPages("ini_test_pages.ini")
	if err != nil {
		t.Fatal("ReadPages:", err)
	}
	// NOTE: Once again, reflect.DeepEqual comes to the rescue :)
	if !reflect.DeepEqual(pages, expected) {
		t.Fatalf("pages differ: expected %#v, got %#v", expected, pages)
	}
}
