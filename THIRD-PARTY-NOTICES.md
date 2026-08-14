# Third-party notices

Sonix is **MIT** (see `LICENSE`). This file records **other people’s** work that we ship or overlay.

Direct npm runtime/dev licences were read from `web/package.json` / installed packages: mostly MIT; TypeScript Apache-2.0; **Inter OFL-1.1** (below). Go module licences (`golang.org/x/crypto`, `golang.org/x/image`, `modernc.org/sqlite` and transitives) are from public proxies with **no private module**; a `go-licenses` report should be recorded when that tool is available.

The Docker image installs Tesseract language data and `poppler-utils`. Publishing **container images** (not just this source) may carry extra redistribution duties — review before pushing images to a registry.

---

## node-hp-scan-to — `deploy/hp-scan/listenCmd.js`

| | |
|---|---|
| **Project** | [node-hp-scan-to](https://github.com/manuc66/node-hp-scan-to) |
| **Author / owner** | Emmanuel Counasse ([manuc66](https://github.com/manuc66)) |
| **Version we overlay** | **1.8.0** (same tag as the digest-pinned `docker.io/manuc66/node-hp-scan-to` image) |
| **Licence** | MIT (see text below) |
| **What Sonix ships** | A **patched copy** of the helper’s `listenCmd` command, bind-mounted over `/app/commands/listenCmd.js`. We do **not** vendor the rest of the helper; Compose pulls the upstream image. |

**Why a copy exists:** stock 1.8.0 writes duplex *back* pages as JPEGs into the inbox and does not flush pending odd pages when the operator switches to the simplex **Sonix** target. The overlay:

1. Lets the back pass inherit `scanToPdf` from the front pass (one merged PDF).
2. Flushes pending odds when switching to simplex.

Those edits are a **derivative work**. MIT allows that if we keep the copyright notice and permission notice — they are in this file and in a header on `listenCmd.js`. Sonix’s changes are offered under the same MIT terms.

**Not affiliated with HP.** Upstream’s own README says the same.

### MIT License (node-hp-scan-to)

```
MIT License

Copyright (c) 2022 Emmanuel Counasse

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

Licence file read from upstream tag `v1.8.0` (same text on `master` as of this note):  
<https://github.com/manuc66/node-hp-scan-to/blob/v1.8.0/LICENSE>

---

## Inter font (`@fontsource/inter`)

Imported in `web/src/main.tsx` and bundled into the SPA, therefore embedded in the Go binary and Docker image. **Reserved Font Name:** Inter. Do not rename the font files.

Copyright (c) 2016 The Inter Project Authors (https://github.com/rsms/inter)

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
http://scripts.sil.org/OFL

```
-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded,
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION AND CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name as
presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.
```

---

## Ollama (not shipped)

Ollama is an **external program** the operator installs. It is not a dependency of this repository. Ollama’s licence and any models you `ollama pull` are your responsibility.
