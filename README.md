[![Go Report Card](https://goreportcard.com/badge/github.com/rjkroege/edwood)](https://goreportcard.com/report/github.com/rjkroege/edwood)[![Build Status](https://travis-ci.com/rjkroege/edwood.svg?branch=master)](https://travis-ci.com/rjkroege/edwood)

# Overview
Go port of Rob Pike's Acme editor. Derived from
[ProjectSerenity](https://github.com/ProjectSerenity/acme) but now
increasingly divergent. ProjectSerenity was itself a transliteration
of the original Acme and libframe C code from
[plan9port](https://9fans.github.io/plan9port/)

Named *edwood* in celebration of the  formative influence of Ed Wood on
Plan9 and the truth of
[ed](http://www.dcs.ed.ac.uk/home/jec/texts/ed.html)-iting.

Note that on unix systems, Edwood (as with Acme) requires some infrastructure from
[plan9port](https://9fans.github.io/plan9port/): in particular
`devdraw`, `9pserve` and `fontsrv`. So to actually use this, you'll want
to install [plan9port](https://9fans.github.io/plan9port/) first.

## Edwood without plan9port

On Windows, plan9port is not required. Work to remove dependency
on plan9port in unix systems is currently in progress (see [issue
#205](https://github.com/rjkroege/edwood/issues/205)).  To use
[duitdraw](https://github.com/ktye/duitdraw) instead of plan9port
`devdraw`, us the `duitdraw` build tag:

	go install -tags duitdraw

Duitdraw can use TTF fonts or compressed Plan 9 bitmap fonts. If the font
name is empty, the [Go Font](https://blog.golang.org/go-fonts) is used.
Example usage:

	edwood -f '' -F '' 	# Use Go font
	edwood -f @12pt -F @12pt	# Go font at 12pt
	edwood -f /usr/share/fonts/TTF/DejaVuSans.ttf@12pt -F /usr/share/fonts/TTF/DejaVuSansMono.ttf@12pt
	edwood -f $PLAN9/font/lucsans/euro.8.font -F $PLAN9/font/lucm/unicode.9.font


# Contributions
Contributions are welcome. Just submit a pull request and we'll review
the code before merging it in.

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
