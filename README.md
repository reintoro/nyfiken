nyfiken
=======
Nyfiken means curious in Swedish. Nyfikend is a daemon which periodically checks for updates on a list of URLs and notifies the user of changes. Nyfikenc is a client which interacts with the daemon.

Installation
------------
    go get github.com/karlek/nyfiken/cmd/nyfikend
    mkdir ~/.config/nyfiken
    cp $GOPATH/src/github.com/karlek/nyfiken/config.ini $GOPATH/src/github.com/karlek/nyfiken/pages.ini ~/.config/nyfiken

Security
--------
#### Warning: there exists some known security plausible scenarios.
If an attacker can modify a nyfiken pages file; nyfiken can be used to:

    - Perform all web-based attacks based on HTTP requests.
    - Scan the network for web-servers or routers and, via site-specific mail-setting, gain access to the information.

Nyfiken(c/d)
------------
Nyfikenc is a client to access the updated information from nyfikend. It can be used to force the program to check all pages again, clear all logged updates and to open them in a browser.

Nyfiken(c/d) communicates on port `5239` by default.

Nyfikenc Usage
--------------
    $ nyfikenc
    Sorry, no updates :(
    $ nyfikenc -f
    Pages will be checked immediately by your demand.
    $ nyfikenc
    http://example.org/
    http...
    $ nyfikenc -r
    Opening all updates with: /usr/bin/browser
    $ nyfikenc -c
    Updates list has been cleared!

API documentation
-----------------
http://godoc.org/github.com/karlek/nyfiken

Public domain
-------------
I hereby release this code into the [public domain](https://creativecommons.org/publicdomain/zero/1.0/).
