package version

import (
	"bytes"
	"encoding/json"
	"runtime"
	"text/template"
)

type (
	Version struct {
		App       string `json:"app"`
		Author    string `json:"author"`
		Version   string `json:"version"`
		GitCommit string `json:"gitCommit"`
		BuildDate string `json:"buildDate"`
		Go        string `json:"go"`
		Arch      string `json:"arch"`
	}
)

const DefaultTitle = `{{.App}} {{.Version}} ({{.GitCommit}}; {{.Go}}/{{.Arch}}; by {{.Author}}; built at {{.BuildDate}})`

func New(opts ...VersionOptionFunc) *Version {
	ver := Version{
		Go:   runtime.Version(),
		Arch: runtime.GOARCH,
	}

	for _, opt := range opts {
		opt(&ver)
	}

	return &ver
}

type VersionOptionFunc func(*Version)

// WithApp sets the app
func WithApp(val string) VersionOptionFunc {
	return func(ver *Version) {
		ver.App = val
	}
}

// WithVersion sets the version
func WithVersion(val string) VersionOptionFunc {
	return func(ver *Version) {
		ver.Version = val
	}
}

// WithGitCommit sets the git commit
func WithGitCommit(val string) VersionOptionFunc {
	return func(ver *Version) {
		ver.GitCommit = val
	}
}

// WithBuildDate sets the build tage
func WithBuildDate(val string) VersionOptionFunc {
	return func(ver *Version) {
		ver.BuildDate = val
	}
}

// WithAuthor sets the author
func WithAuthor(val string) VersionOptionFunc {
	return func(ver *Version) {
		ver.Author = val
	}
}

func (v *Version) SetApp(val string) *Version {
	v.App = val
	return v
}

func (v *Version) SetVersion(val string) *Version {
	v.Version = val
	return v
}

func (v *Version) SetGitCommit(val string) *Version {
	v.GitCommit = val
	return v
}

func (v *Version) SetBuildDate(val string) *Version {
	v.BuildDate = val
	return v
}

func (v *Version) SetAuthor(val string) *Version {
	v.Author = val
	return v
}

func (v *Version) Title(tmpl *string) string {
	var buf bytes.Buffer

	if tmpl == nil {
		tmp := DefaultTitle
		tmpl = &tmp
	}

	// parse template
	t, err := template.New("title").Parse(*tmpl)
	if err != nil {
		panic(err)
	}

	// execute template
	err = t.Execute(&buf, v)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func (v *Version) BuildVersionLine(tmpl *string) (ret string) {
	if tmpl == nil {
		content, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}

		ret = string(content)
	} else {
		var buf bytes.Buffer

		// parse template
		t, err := template.New("version").Parse(*tmpl)
		if err != nil {
			panic(err)
		}

		// execute template
		err = t.Execute(&buf, v)
		if err != nil {
			panic(err)
		}

		ret = buf.String()
	}

	return
}
