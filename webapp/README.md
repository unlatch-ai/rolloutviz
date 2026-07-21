# RLViz browser viewer

This is the static application deployed at `app.rlviz.dev`, separate from the
documentation site at `rlviz.dev`. It reuses the instrument viewer in
`web/src` through the `ViewerProvider` interface and supplies an in-memory Go
WebAssembly provider.

Build from the repository root:

```bash
make web-install
make webapp
python3 -m http.server 8000 --directory webapp/dist
```

`webapp/dist` is a complete static output containing the JavaScript and CSS
bundle, `rlviz.wasm`, and the matching local `wasm_exec.js`. It has no CDN or
runtime package dependency and can be uploaded directly as a Vercel static
output.

Trace files are read through the browser `File` API. Raw trace and adapter bytes
are never sent in a request. The 32 MiB recommended ceiling triggers a warning;
files above 256 MiB are directed to the streaming CLI instead of being loaded
into tab memory.
