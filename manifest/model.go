package manifest

type Model map[string]Metadata

type Metadata struct {
	Root string `json:"root"`
	Path string `json:"path,omitempty"`
	Type string `json:"type"`
}
