package cmd

import (
	"bytes"
	"container/list"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"hash/fnv"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ag5denis/gomarkdoc"
	"github.com/ag5denis/gomarkdoc/format"
	"github.com/ag5denis/gomarkdoc/lang"
	"github.com/ag5denis/gomarkdoc/logger"
)

// Flags populated by goreleaser
var version = ""

const configFilePrefix = ".gomarkdoc"

func BuildCommand() *cobra.Command {
	var opts CommandOptions
	var configFile string

	// cobra.OnInitialize(func() { BuildConfig(configFile) })

	var command = &cobra.Command{
		Use:   "gomarkdoc [package ...]",
		Short: "generate markdown documentation for golang code",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Version {
				PrintVersion()
				return nil
			}

			BuildConfig(configFile)

			// Load configuration from viper
			opts.IncludeUnexported = viper.GetBool("IncludeUnexported")
			opts.Output = viper.GetString("Output")
			opts.Check = viper.GetBool("Check")
			opts.Embed = viper.GetBool("Embed")
			opts.Format = viper.GetString("Format")
			opts.TemplateOverrides = viper.GetStringMapString("template")
			opts.TemplateFileOverrides = viper.GetStringMapString("templateFile")
			opts.Header = viper.GetString("Header")
			opts.HeaderFile = viper.GetString("HeaderFile")
			opts.Footer = viper.GetString("Footer")
			opts.FooterFile = viper.GetString("FooterFile")
			opts.Tags = viper.GetStringSlice("Tags")
			opts.Repository.Remote = viper.GetString("Repository.url")
			opts.Repository.DefaultBranch = viper.GetString("Repository.defaultBranch")
			opts.Repository.PathFromRoot = viper.GetString("Repository.path")

			if opts.Check && opts.Output == "" {
				return errors.New("gomarkdoc: Check mode cannot be run without an Output set")
			}

			if len(args) == 0 {
				// Default to current directory
				args = []string{"."}
			}

			return RunCommand(args, opts)
		},
	}

	command.Flags().StringVar(
		&configFile,
		"config",
		"",
		fmt.Sprintf("File from which to load configuration (default: %s.yml)", configFilePrefix),
	)
	command.Flags().BoolVarP(
		&opts.IncludeUnexported,
		"include-unexported",
		"u",
		false,
		"Output documentation for unexported symbols, methods and fields in addition to exported ones.",
	)
	command.Flags().StringVarP(
		&opts.Output,
		"Output",
		"o",
		"",
		"File or pattern specifying where to write documentation Output. Defaults to printing to stdout.",
	)
	command.Flags().BoolVarP(
		&opts.Check,
		"Check",
		"c",
		false,
		"Check the Output to see if it matches the generated documentation. --Output must be specified to use this.",
	)
	command.Flags().BoolVarP(
		&opts.Embed,
		"Embed",
		"e",
		false,
		"Embed documentation into existing markdown files if available, otherwise append to file.",
	)
	command.Flags().StringVarP(
		&opts.Format,
		"Format",
		"f",
		"github",
		"Format to use for writing Output data. Valid options: github (default), azure-devops, plain",
	)
	command.Flags().StringToStringVarP(
		&opts.TemplateOverrides,
		"template",
		"t",
		map[string]string{},
		"Custom template string to use for the provided template name instead of the default template.",
	)
	command.Flags().StringToStringVar(
		&opts.TemplateFileOverrides,
		"template-file",
		map[string]string{},
		"Custom template file to use for the provided template name instead of the default template.",
	)
	command.Flags().StringVar(
		&opts.Header,
		"Header",
		"",
		"Additional content to inject at the beginning of each Output file.",
	)
	command.Flags().StringVar(
		&opts.HeaderFile,
		"Header-file",
		"",
		"File containing additional content to inject at the beginning of each Output file.",
	)
	command.Flags().StringVar(
		&opts.Footer,
		"Footer",
		"",
		"Additional content to inject at the end of each Output file.",
	)
	command.Flags().StringVar(
		&opts.FooterFile,
		"Footer-file",
		"",
		"File containing additional content to inject at the end of each Output file.",
	)
	command.Flags().StringSliceVar(
		&opts.Tags,
		"Tags",
		DefaultTags(),
		"Set of build Tags to apply when choosing which files to include for documentation generation.",
	)
	command.Flags().CountVarP(
		&opts.Verbosity,
		"verbose",
		"v",
		"Log additional Output from the execution of the command. Can be chained for additional Verbosity.",
	)
	command.Flags().StringVar(
		&opts.Repository.Remote,
		"Repository.url",
		"",
		"Manual override for the git Repository URL used in place of automatic detection.",
	)
	command.Flags().StringVar(
		&opts.Repository.DefaultBranch,
		"Repository.default-branch",
		"",
		"Manual override for the git Repository URL used in place of automatic detection.",
	)
	command.Flags().StringVar(
		&opts.Repository.PathFromRoot,
		"Repository.path",
		"",
		"Manual override for the path from the root of the git Repository used in place of automatic detection.",
	)
	command.Flags().BoolVar(
		&opts.Version,
		"Version",
		false,
		"Print the Version.",
	)

	// We ignore the errors here because they only happen if the specified flag doesn't exist
	_ = viper.BindPFlag("IncludeUnexported", command.Flags().Lookup("include-unexported"))
	_ = viper.BindPFlag("Output", command.Flags().Lookup("Output"))
	_ = viper.BindPFlag("Check", command.Flags().Lookup("Check"))
	_ = viper.BindPFlag("Embed", command.Flags().Lookup("Embed"))
	_ = viper.BindPFlag("Format", command.Flags().Lookup("Format"))
	_ = viper.BindPFlag("template", command.Flags().Lookup("template"))
	_ = viper.BindPFlag("templateFile", command.Flags().Lookup("template-file"))
	_ = viper.BindPFlag("Header", command.Flags().Lookup("Header"))
	_ = viper.BindPFlag("HeaderFile", command.Flags().Lookup("Header-file"))
	_ = viper.BindPFlag("Footer", command.Flags().Lookup("Footer"))
	_ = viper.BindPFlag("FooterFile", command.Flags().Lookup("Footer-file"))
	_ = viper.BindPFlag("Tags", command.Flags().Lookup("Tags"))
	_ = viper.BindPFlag("Repository.url", command.Flags().Lookup("Repository.url"))
	_ = viper.BindPFlag("Repository.defaultBranch", command.Flags().Lookup("Repository.default-branch"))
	_ = viper.BindPFlag("Repository.path", command.Flags().Lookup("Repository.path"))

	return command
}

func DefaultTags() []string {
	f, ok := os.LookupEnv("GOFLAGS")
	if !ok {
		return nil
	}

	fs := flag.NewFlagSet("goflags", flag.ContinueOnError)
	tags := fs.String("Tags", "", "")

	if err := fs.Parse(strings.Fields(f)); err != nil {
		return nil
	}

	if tags == nil {
		return nil
	}

	return strings.Split(*tags, ",")
}

func BuildConfig(configFile string) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName(configFilePrefix)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// TODO: better handling
			fmt.Println(err)
		}
	}
}

func RunCommand(paths []string, opts CommandOptions) error {
	outputTmpl, err := template.New("Output").Parse(opts.Output)
	if err != nil {
		return fmt.Errorf("gomarkdoc: invalid Output template: %w", err)
	}

	specs := GetSpecs(paths...)

	if err := ResolveOutput(specs, outputTmpl); err != nil {
		return err
	}

	if err := LoadPackages(specs, opts); err != nil {
		return err
	}

	return WriteOutput(specs, opts)
}

func ResolveOutput(specs []*PackageSpec, outputTmpl *template.Template) error {
	for _, spec := range specs {
		var outputFile strings.Builder
		if err := outputTmpl.Execute(&outputFile, spec); err != nil {
			return err
		}

		outputStr := outputFile.String()
		if outputStr == "" {
			// Preserve empty values
			spec.OutputFile = ""
		} else {
			// Clean up other values
			spec.OutputFile = filepath.Clean(outputFile.String())
		}
	}

	return nil
}

func ResolveOverrides(opts CommandOptions) ([]gomarkdoc.RendererOption, error) {
	var overrides []gomarkdoc.RendererOption

	// Content overrides take precedence over file overrides
	for name, s := range opts.TemplateOverrides {
		overrides = append(overrides, gomarkdoc.WithTemplateOverride(name, s))
	}

	for name, f := range opts.TemplateFileOverrides {
		// File overrides get applied only if there isn't already a content
		// override.
		if _, ok := opts.TemplateOverrides[name]; ok {
			continue
		}

		b, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("gomarkdoc: couldn't resolve template for %s: %w", name, err)
		}

		overrides = append(overrides, gomarkdoc.WithTemplateOverride(name, string(b)))
	}

	var f format.Format
	switch opts.Format {
	case "github":
		f = &format.GitHubFlavoredMarkdown{}
	case "azure-devops":
		f = &format.AzureDevOpsMarkdown{}
	case "plain":
		f = &format.PlainMarkdown{}
	default:
		return nil, fmt.Errorf("gomarkdoc: invalid Format: %s", opts.Format)
	}

	overrides = append(overrides, gomarkdoc.WithFormat(f))

	return overrides, nil
}

func ResolveHeader(opts CommandOptions) (string, error) {
	if opts.Header != "" {
		return opts.Header, nil
	}

	if opts.HeaderFile != "" {
		b, err := ioutil.ReadFile(opts.HeaderFile)
		if err != nil {
			return "", fmt.Errorf("gomarkdoc: couldn't resolve Header file: %w", err)
		}

		return string(b), nil
	}

	return "", nil
}

func ResolveFooter(opts CommandOptions) (string, error) {
	if opts.Footer != "" {
		return opts.Footer, nil
	}

	if opts.FooterFile != "" {
		b, err := ioutil.ReadFile(opts.FooterFile)
		if err != nil {
			return "", fmt.Errorf("gomarkdoc: couldn't resolve Footer file: %w", err)
		}

		return string(b), nil
	}

	return "", nil
}

func LoadPackages(specs []*PackageSpec, opts CommandOptions) error {
	for _, spec := range specs {
		log := logger.New(GetLogLevel(opts.Verbosity), logger.WithField("dir", spec.Dir))

		buildPkg, err := GetBuildPackage(spec.ImportPath, opts.Tags)
		if err != nil {
			log.Debugf("unable to load package in directory: %s", err)
			// We don't care if a wildcard path produces nothing
			if spec.IsWildcard {
				continue
			}

			return err
		}

		var pkgOpts []lang.PackageOption
		pkgOpts = append(pkgOpts, lang.PackageWithRepositoryOverrides(&opts.Repository))

		if opts.IncludeUnexported {
			pkgOpts = append(pkgOpts, lang.PackageWithUnexportedIncluded())
		}

		pkg, err := lang.NewPackageFromBuild(log, buildPkg, pkgOpts...)
		if err != nil {
			return err
		}

		spec.Pkg = pkg
	}

	return nil
}

func GetBuildPackage(path string, tags []string) (*build.Package, error) {
	ctx := build.Default
	ctx.BuildTags = tags

	if IsLocalPath(path) {
		pkg, err := ctx.ImportDir(path, build.ImportComment)
		if err != nil {
			return nil, fmt.Errorf("gomarkdoc: invalid package in directory: %s", path)
		}

		return pkg, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	pkg, err := ctx.Import(path, wd, build.ImportComment)
	if err != nil {
		return nil, fmt.Errorf("gomarkdoc: invalid package at import path: %s", path)
	}

	return pkg, nil
}

func GetSpecs(paths ...string) []*PackageSpec {
	var expanded []*PackageSpec
	for _, path := range paths {
		// Ensure that the path we're working with is normalized for the OS
		// we're using (i.e. "\" for windows, "/" for everything else)
		path = filepath.FromSlash(path)

		// Not a recursive path
		if !strings.HasSuffix(path, fmt.Sprintf("%s...", string(os.PathSeparator))) {
			isLocal := IsLocalPath(path)
			var dir string
			if isLocal {
				dir = path
			} else {
				dir = "."
			}
			expanded = append(expanded, &PackageSpec{
				Dir:        dir,
				ImportPath: path,
				IsWildcard: false,
				IsLocal:    isLocal,
			})
			continue
		}

		// Remove the recursive marker so we can work with the path
		trimmedPath := path[0 : len(path)-3]

		// Not a file path. Add the original path back to the list so as to not
		// mislead someone into thinking we're processing the recursive path
		if !IsLocalPath(trimmedPath) {
			expanded = append(expanded, &PackageSpec{
				Dir:        ".",
				ImportPath: path,
				IsWildcard: false,
				IsLocal:    false,
			})
			continue
		}

		expanded = append(expanded, &PackageSpec{
			Dir:        trimmedPath,
			ImportPath: trimmedPath,
			IsWildcard: true,
			IsLocal:    true,
		})

		queue := list.New()
		queue.PushBack(trimmedPath)
		for e := queue.Front(); e != nil; e = e.Next() {
			prev := e.Prev()
			if prev != nil {
				queue.Remove(prev)
			}

			p := e.Value.(string)

			files, err := ioutil.ReadDir(p)
			if err != nil {
				// If we couldn't read the folder, there are no directories that
				// we're going to find beneath it
				continue
			}

			for _, f := range files {
				if IsIgnoredDir(f.Name()) {
					continue
				}

				if f.IsDir() {
					subPath := filepath.Join(p, f.Name())

					// Some local paths have their prefixes stripped by Join().
					// If the path is no longer a local path, add the current
					// working directory.
					if !IsLocalPath(subPath) {
						subPath = fmt.Sprintf("%s%s", cwdPathPrefix, subPath)
					}

					expanded = append(expanded, &PackageSpec{
						Dir:        subPath,
						ImportPath: subPath,
						IsWildcard: true,
						IsLocal:    true,
					})
					queue.PushBack(subPath)
				}
			}
		}
	}

	return expanded
}

var ignoredDirs = []string{".git"}

// IsIgnoredDir identifies if the dir is one we want to intentionally ignore.
func IsIgnoredDir(dirname string) bool {
	for _, ignored := range ignoredDirs {
		if ignored == dirname {
			return true
		}
	}

	return false
}

const (
	cwdPathPrefix    = "." + string(os.PathSeparator)
	parentPathPrefix = ".." + string(os.PathSeparator)
)

func IsLocalPath(path string) bool {
	return strings.HasPrefix(path, cwdPathPrefix) || strings.HasPrefix(path, parentPathPrefix) || filepath.IsAbs(path)
}

func Compare(r1, r2 io.Reader) (bool, error) {
	r1Hash := fnv.New128()
	if _, err := io.Copy(r1Hash, r1); err != nil {
		return false, fmt.Errorf("gomarkdoc: failed when checking documentation: %w", err)
	}

	r2Hash := fnv.New128()
	if _, err := io.Copy(r2Hash, r2); err != nil {
		return false, fmt.Errorf("gomarkdoc: failed when checking documentation: %w", err)
	}

	return bytes.Equal(r1Hash.Sum(nil), r2Hash.Sum(nil)), nil
}

func GetLogLevel(verbosity int) logger.Level {
	switch verbosity {
	case 0:
		return logger.WarnLevel
	case 1:
		return logger.InfoLevel
	case 2:
		return logger.DebugLevel
	default:
		return logger.DebugLevel
	}
}

func PrintVersion() {
	if version != "" {
		fmt.Println(version)
		return
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println(info.Main.Version)
	} else {
		fmt.Println("<unknown>")
	}
}
