package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

// The publication API retains each operation-key slot atomically and forever.
// A contiguous, bounded sequence of those slots provides a compare-and-set:
// contenders for the next logical mutation use the SAME key, so the server
// either replays identical intent or refuses a conflict. Never allocate a fresh
// slot inside an uncertain retry, or derive the sequence from desired policy.
// This is a provider protocol over public HTTP receipts, not a native codec.
const publicationSequenceLimit = 64

var publicationManagementKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,23}$`)

func publicationPrefix(kind, key string) (string, error) {
	code := map[string]string{"application": "a", "binding": "b", "target": "t"}[kind]
	if code == "" || !publicationManagementKey.MatchString(key) {
		return "", errors.New("publication management_key requires 1–24 identifier characters")
	}
	return "tf1" + code + "_" + key + "_", nil
}

func publicationKey(kind, key string, sequence int) (string, error) {
	prefix, err := publicationPrefix(kind, key)
	if err != nil {
		return "", err
	}
	if sequence < 0 || sequence >= publicationSequenceLimit {
		return "", errors.New("publication operation capacity exhausted; retained keys cannot be reclaimed")
	}
	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func publicationKeyParts(kind, operationKey string) (string, int, error) {
	separator := strings.LastIndexByte(operationKey, '_')
	if separator < 5 || separator+3 != len(operationKey) {
		return "", 0, errors.New("unsupported publication management identity")
	}
	key := operationKey[5:separator]
	sequence, err := strconv.Atoi(operationKey[separator+1:])
	canonical, keyErr := publicationKey(kind, key, sequence)
	if err != nil || keyErr != nil || canonical != operationKey {
		return "", 0, errors.New("unsupported publication management identity")
	}
	return key, sequence, nil
}

type publicationReceipts interface {
	Capabilities(context.Context) (api.PublicationCapabilities, error)
	Lookup(context.Context, string) (api.PublicationOperation, error)
}

type publicationIdentity struct {
	Owner      string
	Operations []api.PublicationOperation
	// Resource is the latest retained incarnation, even if deleted. Only an
	// authorized exact resource read can establish that it is absent.
	Resource string
}

func publicationAbsent(err error) bool {
	var e *api.PublicationError
	return errors.As(err, &e) && e.Status == 404 && e.Code == "resource_not_found"
}

func publicationPending(op api.PublicationOperation) bool {
	return op.Status == "accepted" || op.Status == "unknown"
}

func publicationInspect(ctx context.Context, client publicationReceipts, kind, key string) (publicationIdentity, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}
	var result publicationIdentity
	if _, err := publicationPrefix(kind, key); err != nil {
		return result, err
	}
	caps, err := client.Capabilities(ctx)
	if err != nil {
		return result, err
	}
	if !caps.Capabilities["operations"] || caps.ReceiptPolicy != "retain-or-tombstone" || caps.Limits["operations"] != publicationSequenceLimit {
		return result, errors.New("publication receipt retention or capacity differs from the qualified protocol")
	}
	result.Owner = caps.PrincipalID
	for sequence := range publicationSequenceLimit {
		operationKey, _ := publicationKey(kind, key, sequence)
		op, err := client.Lookup(ctx, operationKey)
		if publicationAbsent(err) {
			return result, nil
		}
		if err != nil {
			// In particular, 410 is retained uncertainty, never a vacant slot.
			return result, err
		}
		// Wait observationally for the original accepted work. No slot is
		// released or advanced while its outcome remains uncertain.
		for publicationPending(op) {
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return result, errors.New("publication mutation remains unsettled within the configured deadline; retain its original identity")
			case <-timer.C:
			}
			next, err := client.Lookup(ctx, operationKey)
			if err != nil {
				return result, err
			}
			if next.ID != op.ID || next.Key != op.Key || next.Fingerprint != op.Fingerprint || next.PrincipalID != op.PrincipalID || next.Kind != op.Kind || (op.ResourceID != nil && (next.ResourceID == nil || *next.ResourceID != *op.ResourceID)) {
				return result, errors.New("publication settlement changed the retained operation identity")
			}
			op = next
		}
		if op.Key != operationKey || op.PrincipalID != result.Owner || (op.Kind != kind+"_create" && op.Kind != kind+"_update" && op.Kind != kind+"_delete") {
			return result, errors.New("publication management ownership or operation kind conflicts")
		}
		if kind == "binding" && op.Kind == "binding_update" {
			return result, errors.New("unsupported publication binding mutation")
		}
		if op.Kind == kind+"_create" {
			if op.ResourceID != nil {
				result.Resource = *op.ResourceID
			}
		} else if op.ResourceID != nil && *op.ResourceID != result.Resource {
			return result, errors.New("publication receipt refers to another resource incarnation")
		}
		result.Operations = append(result.Operations, op)
	}
	return result, nil
}

func (i publicationIdentity) lastKey() string {
	if len(i.Operations) == 0 {
		return ""
	}
	return i.Operations[len(i.Operations)-1].Key
}

// A mutation is fenced by the last observed slot in the saved Terraform state
// or plan. Re-reading the service must not silently advance that fence.
func publicationSuccessor(kind, key, previous string) (string, error) {
	if previous == "" {
		return publicationKey(kind, key, 0)
	}
	ownerKey, sequence, err := publicationKeyParts(kind, previous)
	if err != nil || ownerKey != key {
		return "", errors.New("publication state has incompatible management identity")
	}
	return publicationKey(kind, key, sequence+1)
}
