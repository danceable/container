package visualize

import (
	"html/template"
	"io"
	"strings"
)

// HTML renders graphs as a standalone web page. The page lays the DOT drawing of the
// graph out in the browser with Viz.js, loaded from a CDN, and falls back to the ASCII
// drawing when scripts do not run.
//
//	http.Handle("/debug/container", c.VisualizeHandler())
type HTML struct {
	// Title is the title of the page, "Dependency graph" when empty.
	Title string
}

func (HTML) ContentType() string {
	return "text/html; charset=utf-8"
}

// Render writes the page to w.
func (h HTML) Render(w io.Writer, g Graph) error {
	var dot, text strings.Builder

	// Writing to a strings.Builder never fails.
	_ = DOT{}.Render(&dot, g)
	_ = ASCII{}.Render(&text, g)

	title := h.Title
	if title == "" {
		title = "Dependency graph"
	}

	return page.Execute(w, struct {
		Title string
		DOT   string
		Text  string
	}{Title: title, DOT: dot.String(), Text: text.String()})
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
	:root { color-scheme: light; --bg: #fafafa; --fg: #202020; --muted: #707070; --panel: #ffffff; --line: #e0e0e0; }
	body { margin: 0; padding: 16px; background: var(--bg); color: var(--fg); font: 14px/1.5 system-ui, sans-serif; }
	h1 { font-size: 18px; font-weight: 600; margin: 0 0 12px; }
	#graph { background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: 16px; overflow: auto; }
	#graph svg { max-width: 100%; height: auto; }
	pre { margin: 0; font: 13px/1.4 ui-monospace, monospace; white-space: pre; }
	details { margin-top: 12px; color: var(--muted); }
	details pre { margin-top: 8px; color: var(--fg); background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: 12px; overflow: auto; }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<div id="graph"><pre>{{.Text}}</pre></div>
<details>
<summary>DOT source</summary>
<pre>{{.DOT}}</pre>
</details>
<script src="https://cdn.jsdelivr.net/npm/@viz-js/viz@3/lib/viz-standalone.js"></script>
<script>
	if (window.Viz) {
		Viz.instance().then(function (viz) {
			var graph = document.getElementById("graph");
			graph.replaceChildren(viz.renderSVGElement({{.DOT}}));
		});
	}
</script>
</body>
</html>
`))
