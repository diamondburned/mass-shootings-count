package frontend

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/diamondburned/tmplutil"
)

//go:embed index static components
var baseFS embed.FS

func init() {
	tmplutil.Log = true
}

var Template = tmplutil.Templater{
	FileSystem: baseFS,
	Includes: map[string]string{
		"styles": "components/styles.html",
		"error":  "components/error.html",
	},
	Functions: template.FuncMap{
		"Plural":  Plural,
		"ToLower": strings.ToLower,
		"ToUpper": strings.ToUpper,
	},
	OnRenderFail: func(sub *tmplutil.Subtemplate, w io.Writer, err error) {
		Template := sub.Templater()
		Template.Subtemplate("error").Execute(w, err)
	},
}

// ErrorHTML writes the error as HTML into w.
func ErrorHTML(w http.ResponseWriter, code int, err error) {
	if code != 0 {
		w.WriteHeader(code)
	}
	Template.Subtemplate("error").Execute(w, err)
}

// OverrideTmpl overrides templates using the given path.
func OverrideTmpl(path string) {
	Template.FileSystem = tmplutil.OverrideFS(baseFS, os.DirFS(path))
}

// Plural returns "($singular | $plural)" depending on n.
func Plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func Preload() {
	Template.Preload()
}

// StaticHandler returns a handler serving files in static/.
func StaticHandler() http.Handler {
	static, err := fs.Sub(Template.FileSystem, "static")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(static))
}
