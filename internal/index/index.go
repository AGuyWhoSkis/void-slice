package index

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Resource struct {
	Name        string // canonical ResourceInfo@name, e.g. models/foo/bar.ext
	RelDeclPath string // path relative to bucket dir, e.g. generated....decl
	RelXMLPath  string // path relative to bucket dir, e.g. generated....decl.xml
}

// xmlWrapper matches: <ResourceInfo name="..." ... />
type xmlWrapper struct {
	Info struct {
		Name string `xml:"name,attr"`
	} `xml:"ResourceInfo"`
}

// Load scans a bucket directory like: Export/game2
// It indexes all *.decl.xml files and maps ResourceInfo@name to the sibling .decl file.
func Load(bucketDir string) (map[string]Resource, error) {
	idx := make(map[string]Resource)

	err := filepath.WalkDir(bucketDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".decl.xml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var w xmlWrapper
		if err := xml.Unmarshal(data, &w); err != nil {
			return nil
		}
		if w.Info.Name == "" {
			return nil
		}

		relXML, err := filepath.Rel(bucketDir, path)
		if err != nil {
			return nil
		}
		relDecl := strings.TrimSuffix(relXML, ".xml")

		if strings.HasPrefix(relXML, "..") || strings.HasPrefix(relDecl, "..") {
			return fmt.Errorf("refused suspicious relative path: %q", relXML)
		}

		idx[w.Info.Name] = Resource{
			Name:        w.Info.Name,
			RelDeclPath: filepath.ToSlash(relDecl),
			RelXMLPath:  filepath.ToSlash(relXML),
		}

		return nil
	})

	return idx, err
}
