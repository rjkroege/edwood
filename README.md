# edwood
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

# Status
Edwood is not yet ready for use but is getting close to being actually useful.
The *useful* milestone will offer the following:

* editing experience effectively identical to Acme
* support for `win` and other filesystem clients
* but... buggy

# Roadmap

* Get to useful.
* More idiomatic Go.
* Fix bugs.
* API modernization

# Build And Test
[![Go Report Card](https://goreportcard.com/badge/github.com/rjkroege/edwood)](https://goreportcard.com/report/github.com/rjkroege/edwood)




