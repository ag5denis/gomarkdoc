package cmd

import "github.com/ag5denis/gomarkdoc/lang"

// PackageSpec defines the data available to the --Output option's template.
// Information is recomputed for each package generated.
type PackageSpec struct {
	// Dir holds the local path where the package is located. If the package is
	// a remote package, this will always be ".".
	Dir string

	// ImportPath holds a representation of the package that should be unique
	// for most purposes. If a package is on the filesystem, this is equivalent
	// to the value of Dir. For remote packages, this holds the string used to
	// import that package in code (e.g. "encoding/json").
	ImportPath string
	IsWildcard bool
	IsLocal    bool
	OutputFile string
	Pkg        *lang.Package
}

type CommandOptions struct {
	Repository            lang.Repo
	Output                string
	Header                string
	HeaderFile            string
	Footer                string
	FooterFile            string
	Format                string
	Tags                  []string
	TemplateOverrides     map[string]string
	TemplateFileOverrides map[string]string
	Verbosity             int
	IncludeUnexported     bool
	Check                 bool
	Embed                 bool
	Version               bool
}
