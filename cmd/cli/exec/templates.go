package exec

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"golang.org/x/exp/maps"
)

//go:embed templates/*.tpl
var embeddedFiles embed.FS

func ErrUnknownTemplate(name string) error {
	return fmt.Errorf("unknown template specified: %s", name)
}

type TemplateMap struct {
	m map[string]string
}

func NewTemplateMap(fsys fs.ReadDirFS, path string) (*TemplateMap, error) {
	entries, err := fsys.ReadDir(path)
	if err != nil {
		return nil, err
	}

	tpl := &TemplateMap{
		m: make(map[string]string),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := nameFromFile(entry.Name())

		fd, err := fsys.Open(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		reader := bufio.NewReader(fd)
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		tpl.m[strings.ToLower(name)] = string(data)
	}

	return tpl, nil
}

func (t *TemplateMap) Get(name string) (string, error) {
	tpl, found := t.m[strings.ToLower(name)]
	if !found {
		return "", ErrUnknownTemplate(name)
	}

	return tpl, nil
}

func (t *TemplateMap) AllTemplates() []string {
	return maps.Keys(t.m)
}

func nameFromFile(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}
