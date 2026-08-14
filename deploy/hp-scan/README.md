# HP scan helper pack

Compose profile `hp-scan` runs pinned `node-hp-scan-to` **1.8.0** with Sonix entrypoint (printer IP from Settings).

Upstream project: [manuc66/node-hp-scan-to](https://github.com/manuc66/node-hp-scan-to) by **Emmanuel Counasse**, **MIT License**. Sonix does not ship the whole helper — only a patched `listenCmd.js` overlay. Full copyright and licence text: [`THIRD-PARTY-NOTICES.md`](../../THIRD-PARTY-NOTICES.md). Not affiliated with HP.

## `listenCmd.js` overlay

Stock **1.8.0** leaves `scanToPdf=false` on the emulated-duplex **back** pass, so page JPEGs land in `DIR` (`/scan/inbox`) instead of `TEMP_DIR`. Sonix then imports each JPEG as its own letter and the helper hits `ENOENT` on merge.

We bind-mount [`listenCmd.js`](listenCmd.js) (image copy + upstream-master fixes; MIT derivative — see the file header):

1. Emulated-duplex **backs** inherit `scanToPdf` / scan identity so pages stay in `TEMP_DIR` until one PDF is written to the inbox.
2. Flush pending odds when the operator switches to **Sonix** (simplex) after a duplex front pass.

**Remove the mount when bumping the image digest past a release that includes those fixes.**
