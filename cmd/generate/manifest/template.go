package manifest

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"unicode"
)

const ManifestName = "manifest.go"

const ManifestTemplate = `
package {{ .PackageName }}

import (
	"embed"
	"resources"
)

// =============== Embedded Directories ===============
{{ range .Directories }}
{{ if .IsEmbedded }}
//go:embed {{ .Name }}
var {{ .Name }}EmbedFS embed.FS
{{ end }}
{{ end }}

// =============== Asset Handles ===============

const (
{{- range .Directories }}
{{- range .Files }}
	{{ .Name }} resources.Asset = {{ .Hash }} // {{ .Path }}
{{- end }}
{{- end }}
)

// ============== Asset Manifests ===============
// TASK: These could possibly be stored as external JSON/YAML files that get loaded at init time

var (
{{- range .Directories }}
	{{ .Name | pascal }}Manifest = resources.AssetManifest{
		{{- range .Files }}
		{{ .Name }}: "{{ .Path }}",
		{{- end }}
	}
{{- end }}
)

// =============== Resource Systems ===============

var (
{{- range .Directories }}	
	{{- if .IsEmbedded }}
	{{ .Name | pascal }} = resources.NewResourceSystem("{{ .Name }}",  {{ .Name | pascal }}Manifest, resources.ResourceSystemOptions{TrimRoot: false})
	{{- else }}
	{{ .Name | pascal }} = resources.NewResourceSystem("{{ .Name }}",  {{ .Name | pascal }}Manifest, resources.ResourceSystemOptions{TrimRoot: true})
	{{- end }}
{{- end }}
)

// =============== Initialization ===============

func init() {
{{- range .Directories }}
	{{- if .IsEmbedded }}
	{{ .Name | pascal }}.UseFilesystem({{ .Name }}EmbedFS)
	{{- end }}
{{- end }}
}
`

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

type Model struct {
	Root        string
	PackageName string
	Directories []ResourceDirectory
}

type ResourceDirectory struct {
	Name         string
	ManifestPath string
	IsEmbedded   bool
	Files        []File
}

type File struct {
	Path string
	Name string
	Hash string
}

func Generate(root, packageName string, embed []string) error {
	model := Model{
		Root:        root,
		PackageName: packageName,
		Directories: make([]ResourceDirectory, 0),
	}

	directories := make(map[string]ResourceDirectory)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		parts := strings.Split(filepath.Clean(relPath), string(filepath.Separator))

		if len(parts) > 0 && parts[0] == "vendor" {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		name := d.Name()

		if name == ManifestName {
			return nil
		}

		rootDir := parts[0]

		directory, ok := directories[rootDir]
		if !ok {
			directory = ResourceDirectory{
				Name:         rootDir,
				ManifestPath: "path",
				IsEmbedded:   contains(embed, rootDir),
				Files:        make([]File, 0),
			}
		}

		fileName := toIdentifier(strings.TrimSuffix(name, filepath.Ext(name)))

		directory.Files = append(directory.Files, File{
			Path: relPath,
			Name: fileName,
			Hash: fmt.Sprintf("0x%x", HashFNV(fileName)),
		})

		directories[rootDir] = directory
		return nil
	})

	if err != nil {
		return err
	}

	for _, dir := range directories {
		slices.SortFunc(dir.Files, func(f1, f2 File) int {
			return strings.Compare(f1.Path, f2.Path)
		})
		model.Directories = append(model.Directories, dir)
	}

	slices.SortFunc(model.Directories, func(d1, d2 ResourceDirectory) int {
		return strings.Compare(d1.Name, d2.Name)
	})

	output := path.Join(root, ManifestName)

	tmpl, err := template.New(ManifestName).
		Funcs(template.FuncMap{
			"contains":     contains,
			"camel":        camel,
			"pascal":       pascal,
			"hashPath":     HashFNV,
			"toIdentifier": toIdentifier,
		}).
		Parse(ManifestTemplate)

	if err != nil {
		return err
	}

	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, model); err != nil {
		return err
	}

	cmd := exec.Command("go", "fmt", output)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func camel(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(string(runes[0]))[0] | 0x20
	return string(runes)
}

func pascal(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(append([]rune{[]rune(s)[0] - 32}, []rune(s)[1:]...))
}
func HashFNV(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func toIdentifier(s string) string {
	// Replace separators with spaces
	cleaned := nonAlnum.ReplaceAllString(s, " ")
	parts := strings.Fields(cleaned)

	if len(parts) == 0 {
		return "_"
	}

	// PascalCase
	for i, p := range parts {
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}

	id := strings.Join(parts, "")

	// If it starts with a digit, prefix underscore
	if unicode.IsDigit([]rune(id)[0]) {
		id = "_" + id
	}

	return id
}
