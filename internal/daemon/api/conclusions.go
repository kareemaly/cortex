package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/architectsession"
	"github.com/kareemaly/cortex/internal/collab"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

type ConclusionHandlers struct {
	deps *Dependencies
}

func NewConclusionHandlers(deps *Dependencies) *ConclusionHandlers {
	return &ConclusionHandlers{deps: deps}
}

type conclusionEntry struct {
	id              string
	conclusionType  string // "architect", "work", or "collab"
	collabID        string
	ticketID        string
	agent           string
	profile         string
	startedAt       time.Time
	concludedAt     time.Time
	rejected        bool
	body            string
	commits         []string
	rejectionReason string
}

func aggregateConclusions(projectPath string, ticketStore *ticket.Store) ([]conclusionEntry, error) {
	var entries []conclusionEntry

	asList, _ := architectsession.List(projectPath)
	for _, c := range asList {
		entries = append(entries, conclusionEntry{
			id:             c.ID,
			conclusionType: "architect",
			agent:          c.Meta.Agent,
			profile:        c.Meta.Profile,
			startedAt:      c.Meta.StartedAt,
			concludedAt:    c.Meta.ConcludedAt,
			body:           c.Body,
		})
	}

	collabList, _ := collab.List(projectPath)
	for _, c := range collabList {
		if c.ConclusionMeta != nil {
			entries = append(entries, conclusionEntry{
				id:             c.ID,
				conclusionType: "collab",
				collabID:       c.ID,
				agent:          c.ConclusionMeta.Agent,
				profile:        c.ConclusionMeta.Profile,
				startedAt:      c.ConclusionMeta.StartedAt,
				concludedAt:    c.ConclusionMeta.ConcludedAt,
				body:           c.ConclusionBody,
			})
		}
	}

	if ticketStore != nil {
		doneTickets, err := ticketStore.List(ticket.StatusDone)
		if err == nil {
			for _, t := range doneTickets {
				if ok, _ := ticketStore.HasConclusion(t.ID); !ok {
					continue
				}
				meta, body, readErr := ticketStore.ReadConclusion(t.ID)
				if readErr != nil {
					continue
				}
				entries = append(entries, conclusionEntry{
					id:              t.ID,
					conclusionType:  "work",
					ticketID:        t.ID,
					agent:           meta.Agent,
					profile:         meta.Profile,
					startedAt:       meta.StartedAt,
					concludedAt:     meta.ConcludedAt,
					rejected:        meta.Rejected,
					body:            body,
					commits:         meta.Commits,
					rejectionReason: meta.RejectionReason,
				})
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].concludedAt.After(entries[j].concludedAt)
	})

	return entries, nil
}

func (h *ConclusionHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	var ticketStore *ticket.Store
	if h.deps.StoreManager != nil {
		if ts, tsErr := h.deps.StoreManager.GetStore(projectPath); tsErr == nil {
			ticketStore = ts
		}
	}

	entries, err := aggregateConclusions(projectPath, ticketStore)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	query := strings.ToLower(r.URL.Query().Get("query"))
	typeFilter := r.URL.Query().Get("type")

	filtered := entries
	if query != "" {
		filtered = entries[:0]
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.body), query) {
				filtered = append(filtered, e)
			}
		}
	}
	if typeFilter != "" {
		var typeFiltered []conclusionEntry
		for _, e := range filtered {
			if e.conclusionType == typeFilter {
				typeFiltered = append(typeFiltered, e)
			}
		}
		filtered = typeFiltered
	}

	total := len(filtered)

	offset := 0
	limit := 0
	if q := r.URL.Query().Get("offset"); q != "" {
		offset, _ = strconv.Atoi(q)
	}
	if q := r.URL.Query().Get("limit"); q != "" {
		limit, _ = strconv.Atoi(q)
	}

	if offset > 0 {
		if offset >= len(filtered) {
			filtered = nil
		} else {
			filtered = filtered[offset:]
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	summaries := make([]types.ConclusionSummary, len(filtered))
	for i, e := range filtered {
		summaries[i] = types.ConclusionSummary{
			ID:          e.id,
			TicketID:    e.ticketID,
			CollabID:    e.collabID,
			Agent:       e.agent,
			Profile:     e.profile,
			StartedAt:   e.startedAt,
			ConcludedAt: e.concludedAt,
			Rejected:    e.rejected,
		}
	}

	writeJSON(w, http.StatusOK, types.ListConclusionsResponse{Conclusions: summaries, Total: total})
}

func (h *ConclusionHandlers) Get(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())

	id := chi.URLParam(r, "id")

	asConc, asErr := architectsession.ReadConclusion(projectPath, id)
	if asErr == nil && asConc != nil {
		resp := types.ConclusionResponse{
			ID:          asConc.ID,
			Agent:       asConc.Meta.Agent,
			Profile:     asConc.Meta.Profile,
			Body:        asConc.Body,
			StartedAt:   asConc.Meta.StartedAt,
			ConcludedAt: asConc.Meta.ConcludedAt,
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	collabConc, _, collabErr := collab.ReadConclusion(projectPath, id)
	if collabErr == nil && collabConc != nil {
		resp := types.ConclusionResponse{
			ID:          id,
			CollabID:    id,
			Agent:       collabConc.Agent,
			Profile:     collabConc.Profile,
			Body:        "",
			StartedAt:   collabConc.StartedAt,
			ConcludedAt: collabConc.ConcludedAt,
		}
		fullCollab, _ := collab.ReadPrompt(projectPath, id)
		if fullCollab != nil && fullCollab.ConclusionMeta != nil {
			resp.Body = fullCollab.ConclusionBody
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if h.deps.StoreManager != nil {
		store, err := h.deps.StoreManager.GetStore(projectPath)
		if err == nil {
			hasConc, _ := store.HasConclusion(id)
			if hasConc {
				meta, body, readErr := store.ReadConclusion(id)
				if readErr == nil {
					resp := types.ConclusionResponse{
						ID:              id,
						TicketID:        id,
						Agent:           meta.Agent,
						Profile:         meta.Profile,
						Body:            body,
						Commits:         meta.Commits,
						Rejected:        meta.Rejected,
						RejectionReason: meta.RejectionReason,
						StartedAt:       meta.StartedAt,
						ConcludedAt:     meta.ConcludedAt,
					}
					writeJSON(w, http.StatusOK, resp)
					return
				}
			}
		}
	}

	writeError(w, http.StatusNotFound, "not_found", "conclusion not found")
}

func (h *ConclusionHandlers) Show(w http.ResponseWriter, r *http.Request) {
	projectPath := GetArchitectPath(r.Context())
	id := chi.URLParam(r, "id")

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	if err := openCortexPopup(projectPath, h.deps.TmuxManager, "conclusion", "show", id); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Conclusion viewer opened",
	})
}
