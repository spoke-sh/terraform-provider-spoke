package platformclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
)

// History contains bounded, complete metadata. Payloads and execution outputs
// remain with their independent data owners, outside Terraform state.
type HistoryItem struct {
	ID        string `json:"id"`
	SourceID  string `json:"source_id"`
	Revision  int64  `json:"revision"`
	Timestamp int64  `json:"timestamp"`
	State     string `json:"state"`
	Attempt   int64  `json:"attempt"`
}
type History struct {
	Scope
	ParentID string        `json:"parent_id"`
	Kind     string        `json:"kind"`
	Complete bool          `json:"complete"`
	Items    []HistoryItem `json:"items"`
}

func (h *History) UnmarshalJSON(raw []byte) error {
	if err := requireMembers(raw, reflect.TypeOf(History{})); err != nil {
		return err
	}
	type plain History
	var v plain
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return err
	}
	var members struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(raw, &members)
	for _, i := range members.Items {
		if err := requireMembers(i, reflect.TypeOf(HistoryItem{})); err != nil {
			return err
		}
	}
	for _, item := range v.Items {
		if !ValidID(item.ID) || !ValidID(item.SourceID) || item.Revision < 1 || item.Timestamp < 1 {
			return errors.New("invalid history item")
		}
	}
	*h = History(v)
	return nil
}
func (c *Client) History(ctx context.Context, kind, parent string) ([]HistoryItem, error) {
	if !slices.Contains([]string{"job", "backup_policy"}, kind) || !c.scope.AllowsKind(kind) || !ValidID(parent) {
		return nil, errors.New("history requires an exact scoped parent")
	}
	suffix := "executions"
	if kind == "backup_policy" {
		suffix = "recovery_points"
	}
	var h History
	if err := c.request(ctx, "GET", c.base()+"/"+Collection(kind)+"/"+parent+"/"+suffix, nil, &h); err != nil {
		return nil, err
	}
	if h.Scope != c.scope || h.Kind != kind || h.ParentID != parent || !h.Complete || h.Items == nil || len(h.Items) > min(c.maxItems, 2048) {
		return nil, errors.New("incomplete or unbound history")
	}
	previous := ""
	for _, item := range h.Items {
		if item.ID <= previous || kind == "job" && (!slices.Contains([]string{"active", "unknown", "succeeded", "failed"}, item.State) || item.Attempt < 1 || item.Attempt > 11) || kind == "backup_policy" && (item.State != "complete" || item.Attempt != 0) {
			return nil, errors.New("invalid history outcome")
		}
		previous = item.ID
	}
	return h.Items, nil
}
