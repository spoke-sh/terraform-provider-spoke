package mockplatform

import (
	"crypto/rand"
	"net/http"
	"slices"
	"strings"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

type cursor struct {
	Scope                 api.Scope
	Kind, Principal, Last string
	Epoch                 int64
}

func (s *Server) list(w http.ResponseWriter, r *http.Request, p Principal, scope api.Scope, kind string) {
	query := r.URL.Query()
	if len(query) > 1 || (len(query) == 1 && (len(query["page_token"]) != 1 || query.Get("page_token") == "")) {
		fail(w, 400, "invalid_request")
		return
	}
	token := query.Get("page_token")
	last := ""
	if token != "" {
		c, ok := s.cursors[token]
		if !ok || c.Scope != scope || c.Kind != kind || c.Principal != p.ID || c.Epoch != s.state.Epoch {
			fail(w, 409, "incomplete_result")
			return
		}
		last = c.Last
	}
	keys := []string{}
	for path, o := range s.state.Objects {
		if o.Scope == scope && o.ResourceKind == kind && strings.Compare(o.ID, last) > 0 {
			keys = append(keys, path)
		}
	}
	slices.Sort(keys)
	page := api.Page{Scope: scope, ResourceKind: kind, Items: []api.Object{}, Snapshot: s.state.Epoch, Complete: true}
	const pageSize = 2
	for i, key := range keys {
		if i == pageSize {
			page.Complete = false
			page.NextToken = rand.Text()
			if len(s.cursors) >= 1024 {
				s.cursors = map[string]cursor{}
			}
			s.cursors[page.NextToken] = cursor{scope, kind, p.ID, page.Items[len(page.Items)-1].ID, s.state.Epoch}
			break
		}
		page.Items = append(page.Items, s.state.Objects[key])
	}
	send(w, 200, page)
}
