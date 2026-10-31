package provider

import (
	"errors"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	api "github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
	"strings"
)

func newBucketResource() resource.Resource { return &scopeResource{kind: "bucket"} }

func canonicalID(scope api.Scope, id string) string {
	return strings.Join([]string{scope.OrganizationID, scope.ProjectID, scope.EnvironmentID, id}, "/")
}
func parseID(scope api.Scope, id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return "", errors.New("bucket ID must be organization_id/project_id/environment_id/bucket_id")
	}
	for _, part := range parts {
		if !api.ValidID(part) {
			return "", errors.New("bucket ID contains an invalid scope or object identifier")
		}
	}
	if parts[0] != scope.OrganizationID || parts[1] != scope.ProjectID || parts[2] != scope.EnvironmentID {
		return "", errors.New("bucket identity does not match the configured organization/project/environment")
	}
	return parts[3], nil
}
