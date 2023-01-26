package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/princjef/gomarkdoc"
	"github.com/princjef/gomarkdoc/lang"
	"github.com/princjef/gomarkdoc/logger"
)

// WriteOutput writes the Output of the documentation to the specified files.
func WriteOutput(specs []*PackageSpec, opts CommandOptions) error {
	log := logger.New(GetLogLevel(opts.Verbosity))

	overrides, err := ResolveOverrides(opts)
	if err != nil {
		return err
	}

	out, err := gomarkdoc.NewRenderer(overrides...)
	if err != nil {
		return err
	}

	header, err := ResolveHeader(opts)
	if err != nil {
		return err
	}

	footer, err := ResolveFooter(opts)
	if err != nil {
		return err
	}

	filePkgs := make(map[string][]*lang.Package)

	for _, spec := range specs {
		if spec.Pkg == nil {
			continue
		}

		filePkgs[spec.OutputFile] = append(filePkgs[spec.OutputFile], spec.Pkg)
	}

	for fileName, pkgs := range filePkgs {
		file := lang.NewFile(header, footer, pkgs)

		text, err := out.File(file)
		if err != nil {
			return err
		}

		if opts.Embed && fileName != "" {
			text = EmbedContents(log, fileName, text)
		}

		switch {
		case fileName == "":
			fmt.Fprint(os.Stdout, text)
		case opts.Check:
			var b bytes.Buffer
			fmt.Fprint(&b, text)
			if err := CheckFile(&b, fileName); err != nil {
				return err
			}
		default:
			if err := WriteFile(fileName, text); err != nil {
				return fmt.Errorf("failed to write Output file %s: %w", fileName, err)
			}
		}
	}

	return nil
}

// WriteFile writes the specified text to the specified file.
func WriteFile(fileName string, text string) error {
	folder := filepath.Dir(fileName)

	if folder != "" {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
	}

	if err := ioutil.WriteFile(fileName, []byte(text), 0664); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fileName, err)
	}

	return nil
}

func CheckFile(b *bytes.Buffer, path string) error {
	checkErr := errors.New("Output does not match current files. Did you forget to run gomarkdoc?")

	f, err := os.Open(path)
	if err != nil {
		if err == os.ErrNotExist {
			return checkErr
		}

		return fmt.Errorf("failed to open file %s for checking: %w", path, err)
	}

	defer f.Close()

	match, err := Compare(b, f)
	if err != nil {
		return fmt.Errorf("failure while attempting to Check contents of %s: %w", path, err)
	}

	if !match {
		return checkErr
	}

	return nil
}

var (
	embedStandaloneRegex = regexp.MustCompile(`(?m:^ *)<!--\s*gomarkdoc:Embed\s*-->(?m:\s*?$)`)
	embedStartRegex      = regexp.MustCompile(
		`(?m:^ *)<!--\s*gomarkdoc:Embed:start\s*-->(?s:.*?)<!--\s*gomarkdoc:Embed:end\s*-->(?m:\s*?$)`,
	)
)

func EmbedContents(log logger.Logger, fileName string, text string) string {
	embedText := fmt.Sprintf("<!-- gomarkdoc:Embed:start -->\n\n%s\n\n<!-- gomarkdoc:Embed:end -->", text)

	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Debugf("unable to find Output file %s for embedding. Creating a new file instead", fileName)
		return embedText
	}

	var replacements int
	data = embedStandaloneRegex.ReplaceAllFunc(data, func(_ []byte) []byte {
		replacements++
		return []byte(embedText)
	})

	data = embedStartRegex.ReplaceAllFunc(data, func(_ []byte) []byte {
		replacements++
		return []byte(embedText)
	})

	if replacements == 0 {
		log.Debugf("no Embed markers found. Appending documentation to the end of the file instead")
		return fmt.Sprintf("%s\n\n%s", string(data), text)
	}

	return string(data)
}
