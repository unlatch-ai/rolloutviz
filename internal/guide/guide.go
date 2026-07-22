// Package guide embeds the user documentation shared by every RLViz surface.
package guide

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed pages/*.md
var files embed.FS

var orderedPages = []string{"overview", "install", "workspace", "formats", "agents", "privacy"}

// Markdown is the complete version-matched documentation for CLI and agents.
var Markdown = func() string {
	var output strings.Builder
	for index, name := range orderedPages {
		content, err := files.ReadFile("pages/" + name + ".md")
		if err != nil {
			panic(fmt.Sprintf("embed guide page %s: %v", name, err))
		}
		if index > 0 {
			output.WriteString("\n---\n\n")
		}
		output.Write(content)
		if len(content) == 0 || content[len(content)-1] != '\n' {
			output.WriteByte('\n')
		}
	}
	return output.String()
}()
