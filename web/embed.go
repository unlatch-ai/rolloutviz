// Package webassets exposes the compiled RolloutViz viewer for embedding in the
// release binary. Run `npm --prefix web run build` after changing the UI.
package webassets

import "embed"

// Dist contains the Vite production build.
//
//go:embed dist
var Dist embed.FS
