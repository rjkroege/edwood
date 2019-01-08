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

Note that Edwood (as with Acme) requires some infrastructure from
[plan9port](https://9fans.github.io/plan9port/): in particular
`devdraw` and the p9p font server. So to actually use this, you'll want
to install [plan9port](https://9fans.github.io/plan9port/) first.

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




