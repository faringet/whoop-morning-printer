package art

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed templates/*.txt
var embeddedTemplatesFS embed.FS

func LoadEmbeddedTemplates() ([]Template, error) {
	entries, err := fs.ReadDir(embeddedTemplatesFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("art: read embedded templates dir: %w", err)
	}

	templates := make([]Template, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !strings.EqualFold(path.Ext(fileName), ".txt") {
			continue
		}

		body, err := embeddedTemplatesFS.ReadFile("templates/" + fileName)
		if err != nil {
			return nil, fmt.Errorf("art: read embedded template %q: %w", fileName, err)
		}

		name := strings.TrimSuffix(fileName, path.Ext(fileName))
		name = strings.TrimSpace(name)

		text := strings.TrimSpace(string(body))
		if name == "" || text == "" {
			continue
		}

		templates = append(templates, Template{
			Name: name,
			Body: text,
		})
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	if len(templates) == 0 {
		return nil, fmt.Errorf("art: no embedded templates found")
	}

	return templates, nil
}
