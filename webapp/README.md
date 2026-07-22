# RLViz browser viewer

This package builds the browser entry point served at `rlviz.dev`. It reuses
the instrument viewer in `web/src` through `ViewerProvider` and supplies an
in-memory Go WebAssembly provider. The generated documentation pages are added
to the same deployment by `make site`.

Build from the repository root:

```bash
make site
python3 -m http.server 8000 --directory site/dist
```

The output has no CDN or runtime package dependency. Trace files are read with
the browser `File` API and never sent in a request. The Go core and UI enforce
the same 32 MiB ceiling; larger files are directed to the streaming CLI.

The content policy permits only bundled scripts, workers, styles, images, and
the Go WebAssembly runtime. It allows no remote origin and denies framing,
referrers, MIME sniffing, and non-local form actions.
