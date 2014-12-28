// Package ini retrieves configuration for nyfiken programs from INI files.
package ini

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jteeuwen/ini"
	"github.com/karlek/nyfiken/page"
	"github.com/karlek/nyfiken/settings"
	"github.com/mewkiz/pkg/errutil"
)

// INI sections (i.e. [sectionName]).
const (
	sectionSettings = "settings"
	sectionMail     = "mail"
)

// INI field names.
const (
	fieldBrowser        = "browser"
	fieldFilePerms      = "fileperms"
	fieldHeader         = "header"
	fieldInterval       = "interval"
	fieldNegexp         = "negexp"
	fieldPortNum        = "portnum"
	fieldRecvMail       = "recvmail"
	fieldRegexp         = "regexp"
	fieldSelection      = "sel"
	fieldSendAuthServer = "sendauthserver"
	fieldSendMail       = "sendmail"
	fieldSendOutServer  = "sendoutserver"
	fieldSendPass       = "sendpass"
	fieldSleepStart     = "sleepstart"
	fieldStrip          = "strip"
	fieldThreshold      = "threshold"
)

var (
	// Valid fields in different sections
	siteFields = map[string]bool{
		fieldInterval:  true,
		fieldStrip:     true,
		fieldRecvMail:  true,
		fieldSelection: true,
		fieldRegexp:    true,
		fieldNegexp:    true,
		fieldThreshold: true,
		fieldHeader:    true,
	}
	mailFields = map[string]bool{
		fieldRecvMail:       true,
		fieldSendMail:       true,
		fieldSendPass:       true,
		fieldSendAuthServer: true,
		fieldSendOutServer:  true,
	}
	settingsFields = map[string]bool{
		fieldInterval:  true,
		fieldBrowser:   true,
		fieldPortNum:   true,
		fieldFilePerms: true,
	}
)

// Error messages.
var (
	errFieldNotExist          = "ini: field `%s` doesn't exist."
	errNoSectionSettings      = "ini: no [" + sectionSettings + "] section found config.ini."
	errNoSectionMail          = "ini: no [" + sectionMail + "] section found in config.ini."
	errInvalidMailAddress     = "ini: invalid mail: `%s`; correct syntax -> `name@domain.tld`."
	errInvalidHeader          = "ini: invalid header: `%s`; correct syntax -> `HeaderName: Value`."
	errInvalidStripFunction   = "ini: invalid strip function: `%s`."
	errInvalidRandInterval    = "ini: invalid random interval: %s; correct syntax -> `duration duration`."
	errMailAddressNotFound    = "ini: global receiving mail required."
	errMailAuthServerNotFound = "ini: sending mail authorization server required."
	errMailOutServerNotFound  = "ini: sending mail outgoing server required."
	errInvalidListDeclaration = "ini: use `<` instead of `=` for list values."
)

// Whitelist of allowed strip functions.
var (
	stripFunctions = map[string]bool{
		"html":    true,
		"attrs":   true,
		"numbers": true,
		"scripts": true,
	}
)

// ReadIni is a convenience function wrapping ReadSettings and ReadPages.
func ReadIni(configPath, pagesPath string) (pages []*page.Page, err error) {
	// Read config.
	err = ReadSettings(configPath)
	if err != nil {
		return nil, errutil.Err(err)
	}

	// Read pages file.
	pages, err = ReadPages(pagesPath)
	if err != nil {
		return nil, errutil.Err(err)
	}

	return pages, nil
}

// ReadSettings reads settings file and updates settings.Global.
func ReadSettings(configPath string) (err error) {
	// Parse config file.
	file := ini.New()
	err = file.Load(configPath)
	if err != nil {
		return errutil.Err(err)
	}

	config, settingExist := file.Sections[sectionSettings]
	mail, mailExist := file.Sections[sectionMail]
	if settingExist {
		err = parseSettings(config)
		if err != nil {
			return errutil.Err(err)
		}
	}
	if mailExist {
		err = parseMail(mail)
		if err != nil {
			return errutil.Err(err)
		}
	}

	return nil
}

// Parse ini settings section to global setting.
func parseSettings(config ini.Section) (err error) {
	for fieldName := range config {
		if _, found := settingsFields[fieldName]; !found {
			return errutil.NewNoPosf(errFieldNotExist, fieldName)
		}
	}

	// Get time setting from INI.
	// If interval setting wasn't found, default value is 1 minute
	intervalStr := config.S(fieldInterval, settings.DefaultInterval.String())
	// NOTE: As settings.Global is used several times you could make use of a
	// local pointer to it, thus making the lines shorted and more readable. I've
	// included this change in parseSettings and parseMail but if it seems ugly
	// to you, just revert it.
	global := &settings.Global
	// Parse string to duration.
	global.Interval, err = time.ParseDuration(intervalStr)
	if err != nil {
		return errutil.Err(err)
	}

	// Set global file permissions.
	global.FilePerms = os.FileMode(config.I(fieldFilePerms, int(settings.DefaultFilePerms)))

	// Set port number.
	global.PortNum = config.S(fieldPortNum, settings.DefaultPortNum)

	// Set browser path.
	global.Browser = config.S(fieldBrowser, "")

	return nil
}

// Parse ini mail section to global setting.
func parseMail(mail ini.Section) (err error) {
	for fieldName := range mail {
		if _, found := mailFields[fieldName]; !found {
			return errutil.NewNoPosf(errFieldNotExist, fieldName)
		}
	}

	// Set global sender mail.
	global := &settings.Global
	global.SenderMail.Address = mail.S(fieldSendMail, "")
	if global.SenderMail.Address == "" {
		return errutil.NewNoPosf(errMailAddressNotFound)
	} else if !strings.Contains(global.SenderMail.Address, "@") {
		return errutil.NewNoPosf(errInvalidMailAddress, global.SenderMail.Address)
	}

	// Set global sender mail password.
	global.SenderMail.Password = mail.S(fieldSendPass, "")

	// Set global sender authorization server.
	global.SenderMail.AuthServer = mail.S(fieldSendAuthServer, "")
	if global.SenderMail.AuthServer == "" {
		return errutil.NewNoPosf(errMailAuthServerNotFound)
	}

	// Set global sender mail outgoing server.
	global.SenderMail.OutServer = mail.S(fieldSendOutServer, "")
	if global.SenderMail.OutServer == "" {
		return errutil.NewNoPosf(errMailOutServerNotFound)
	}

	// Set global receive mail.
	global.RecvMail = mail.S(fieldRecvMail, "")
	if global.RecvMail == "" {
		return errutil.NewNoPosf(errMailAddressNotFound)
	} else if !strings.Contains(global.RecvMail, "@") {
		return errutil.NewNoPosf(errInvalidMailAddress, global.RecvMail)
	}

	return nil
}

// ReadPages reads pages file and returns a slice of pages.
func ReadPages(pagesPath string) (pages []*page.Page, err error) {

	// Parse pages file.
	file := ini.New()
	err = file.Load(pagesPath)
	if err != nil {
		return nil, errutil.Err(err)
	}

	// Loop through the INI sections ([section]) and parse page settings.
	for name, section := range file.Sections {
		// Skip global scope INI values since they are empty.
		if len(name) == 0 {
			continue
		}

		if settings.Verbose {
			fmt.Println("[o] Watching:", name)
		}

		for fieldName := range section {
			if _, found := siteFields[fieldName]; !found {
				return nil, errutil.NewNoPosf(errFieldNotExist, fieldName)
			}
		}

		var p page.Page
		var pageSettings settings.Page

		// Make INI section ([http://example.org]) into url.URL.
		p.ReqUrl, err = url.Parse(name)
		if err != nil {
			return nil, errutil.Err(err)
		}

		// Set CSS selector.
		pageSettings.Selection = section.S(fieldSelection, "")

		// Set regular expression string.
		pageSettings.Regexp = section.S(fieldRegexp, "")

		// Set "negexp" (negative regular expression) string which removes all
		// that matches it.
		pageSettings.Negexp = section.S(fieldNegexp, "")

		// Set threshold value.
		pageSettings.Threshold = section.F64(fieldThreshold, 0)

		// Set interval time.
		intervalStr := section.S(fieldInterval, settings.Global.Interval.String())
		// Parse string to duration.
		pageSettings.Interval, err = time.ParseDuration(intervalStr)
		if err != nil {
			return nil, errutil.Err(err)
		}

		// Set individual mail address.
		pageSettings.RecvMail = section.S(fieldRecvMail, settings.Global.RecvMail)
		if pageSettings.RecvMail != "" && !strings.Contains(pageSettings.RecvMail, "@") {
			return nil, errutil.NewNoPosf(errInvalidMailAddress, pageSettings.RecvMail)
		}

		// Set individual header.
		headers := section.List(fieldHeader)
		m := make(map[string]string)
		for _, header := range headers {
			if strings.Contains(header, ":") {
				keyVal := strings.SplitN(header, ":", 2)
				m[strings.TrimSpace(keyVal[0])] = strings.TrimSpace(keyVal[1])
			} else {
				return nil, errutil.NewNoPosf(errInvalidHeader, header)
			}
		}
		pageSettings.Header = m

		// Set strip functions to use.
		pageSettings.StripFuncs = section.List(fieldStrip)
		if pageSettings.StripFuncs == nil {
			if _, found := section[fieldStrip]; found {
				return nil, errutil.NewNoPosf(errInvalidListDeclaration)
			}
		}
		for _, stripFunc := range pageSettings.StripFuncs {
			if _, found := stripFunctions[stripFunc]; !found {
				return nil, errutil.NewNoPosf(errInvalidStripFunction, stripFunc)
			}
		}
		p.Settings = pageSettings

		pages = append(pages, &p)
	}

	if pages == nil {
		return nil, errutil.NewNoPosf("no pages in %s", settings.PagesPath)
	}
	return pages, nil
}
