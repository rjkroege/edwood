[![Go Report Card](https://goreportcard.com/badge/github.com/rjkroege/edwood)](https://goreportcard.com/report/github.com/rjkroege/edwood)[![Build Status](https://github.com/rjkroege/edwood/actions/workflows/edwood.yml/badge.svg?branch=master)](https://github.com/rjkroege/edwood/actions)

# Overview
Go port of Rob Pike's Acme editor. Derived from
[ProjectSerenity](https://github.com/ProjectSerenity/acme) but now
increasingly divergent. ProjectSerenity was itself a transliteration
of the original Acme and libframe C code from
[plan9port](https://9fans.github.io/plan9port/)

Named *edwood* in celebration of the  formative influence of Ed Wood on
Plan9 and the truth of
[ed](http://www.dcs.ed.ac.uk/home/jec/texts/ed.html)-iting.

Note that on unix systems, Edwood (as with Acme) requires by default some
infrastructure from [plan9port](https://9fans.github.io/plan9port/):
in particular `devdraw`, `9pserve` and `fontsrv`. (Note that many other
utilities like `win` and `9pfuse` that contribute to Edwood's utility
are also found in [plan9port](https://9fans.github.io/plan9port/).) So, you'll want to
install [plan9port](https://9fans.github.io/plan9port/) first, unless
you choose to use the more experimental pure-Go Edwood described below.

## Edwood without plan9port

On Windows, plan9port is never used. On unix systems, plan9port is not
used only when the `duitdraw` and `mux9p` tags are used:

	go get -u -tags 'duitdraw mux9p' github.com/rjkroege/edwood

These tags replaces `devdraw` with
[duitdraw](https://github.com/ktye/duitdraw) and `9pserve` with
[mux9p](https://github.com/fhs/mux9p). Note that there are several
outstanding [issues](https://github.com/rjkroege/edwood/issues/205)
which makes Edwood more unstable and slower when not using plan9port.

Duitdraw can use TTF fonts or compressed Plan 9 bitmap fonts. If the font
name is empty, the [Go Font](https://blog.golang.org/go-fonts) is used.
Example usage:

	edwood	# Use Go font at 10pt
	edwood -f @12pt -F @12pt	# Go font at 12pt
	edwood -f /usr/share/fonts/TTF/DejaVuSans.ttf@12pt -F /usr/share/fonts/TTF/DejaVuSansMono.ttf@12pt
	edwood -f $PLAN9/font/lucsans/euro.8.font -F $PLAN9/font/lucm/unicode.9.font

## Edwood on Plan 9

To build Edwood on Plan 9, use [9fans.net/go PR#28](https://github.com/9fans/go/pull/28):

	hget https://github.com/rjkroege/edwood/archive/master.tar.gz | tar xvz
	cd edwood-master
	go mod edit -replace '9fans.net/go=github.com/fhs/9fans-go@plan9-pr'
	go build
	./edwood

# Contributions
Contributions are welcome. Just submit a pull request and we'll review
the code before merging it in.

# Discussion
Have thoughts? Questions? Want to talk about Edwood (or Acme and other related Plan9 things)? If so, I've now enabled [Edwood GitHub Discussions](https://github.com/rjkroege/edwood/discussions).

# Project Status
Edwood has reached the *useful* milestone (v0.1) and should
serve as drop-in replacement for Plan9 Port Acme. (But probably with
different bugs.) Please file issues if Acme client apps don't work
with Edwood or if your favourite Acme feature doesn't work.

# Roadmap

* More idiomatic Go and tests.
* Internal API modernization.
* Revised text handling data structures.
* More configurability: styles, keyboard shortcuts, autocomplete.
* See the issues list for the details.
* Improve the testing [code coverage](https://codecov.io/gh/rjkroege/edwood)
