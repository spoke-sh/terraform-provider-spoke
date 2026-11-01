package mockplatform

import (
	"cmp"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"net/http"
	"slices"
)

func (s *Server) readHistory(w http.ResponseWriter, scope api.Scope, kind, parent string) {
	if !scope.AllowsKind(kind) || !api.ValidID(parent) {
		fail(w, 400, "invalid_request")
		return
	}

	h := api.History{Scope: scope, Kind: kind, ParentID: parent, Complete: true, Items: []api.HistoryItem{}}
	if kind == "job" {
		for _, r := range s.state.Lifecycle.Runs {
			if r.Scope == scope && r.Job == parent {
				h.Items = append(h.Items, api.HistoryItem{ID: r.ID, SourceID: r.Schedule, Revision: r.Revision, Timestamp: r.Due, State: r.State, Attempt: r.Attempt})
			}
		}
	} else {
		for _, p := range s.state.Lifecycle.Points {
			if p.Scope == scope && p.Policy == parent && p.Complete {
				h.Items = append(h.Items, api.HistoryItem{ID: p.ID, SourceID: p.Database, Revision: p.Revision, Timestamp: p.Created, State: "complete"})
			}
		}
	}
	if len(h.Items) == 0 && s.state.Objects[objectPath(scope, kind, parent)].ID == "" {
		fail(w, 404, "resource_not_found")
		return
	}
	slices.SortFunc(h.Items, func(a, b api.HistoryItem) int { return cmp.Compare(a.ID, b.ID) })
	send(w, 200, h)
}
