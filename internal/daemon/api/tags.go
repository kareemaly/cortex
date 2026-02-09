package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

// TagsHandler returns a handler for GET /tags.
// It aggregates tags from both tickets and docs, returning them sorted by count descending.
func TagsHandler(storeManager *StoreManager, docsStoreManager *DocsStoreManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectPath := GetProjectPath(r.Context())

		counts := make(map[string]int)

		// Aggregate ticket tags
		if storeManager != nil {
			store, err := storeManager.GetStore(projectPath)
			if err == nil {
				allTickets, err := store.ListAll()
				if err == nil {
					for _, tickets := range allTickets {
						for _, t := range tickets {
							for _, tag := range t.Tags {
								counts[strings.ToLower(tag)]++
							}
						}
					}
				}
			}
		}

		// Aggregate doc tags
		if docsStoreManager != nil {
			docsStore, err := docsStoreManager.GetStore(projectPath)
			if err == nil {
				allDocs, err := docsStore.List("", "", "")
				if err == nil {
					for _, d := range allDocs {
						for _, tag := range d.Tags {
							counts[strings.ToLower(tag)]++
						}
					}
				}
			}
		}

		// Build sorted slice
		tags := make([]TagCount, 0, len(counts))
		for name, count := range counts {
			tags = append(tags, TagCount{Name: name, Count: count})
		}
		sort.Slice(tags, func(i, j int) bool {
			if tags[i].Count != tags[j].Count {
				return tags[i].Count > tags[j].Count
			}
			return tags[i].Name < tags[j].Name
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ListTagsResponse{Tags: tags}); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to encode response")
		}
	}
}
