// Package provider contains the experimental Spoke Terraform provider.
package provider

import (
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/spoke-sh/terraform-provider-spoke/internal/platformclient"
)

var _ provider.Provider = (*spokeProvider)(nil)

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &spokeProvider{version: version} }
}

type spokeProvider struct{ version string }

type providerModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ProjectID      types.String `tfsdk:"project_id"`
	EnvironmentID  types.String `tfsdk:"environment_id"`
	TokenFile      types.String `tfsdk:"token_file"`
	CAFile         types.String `tfsdk:"ca_file"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
}

type configured struct {
	client  *platformclient.Client
	timeout time.Duration
}

func idValidators() []validator.String {
	return []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`), "must be an opaque identifier with 1–128 letters, digits, underscores, or hyphens, starting with a letter or digit")}
}

func (p *spokeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spoke"
	resp.Version = p.version
}

func (p *spokeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Experimental Spoke provider for the provisional `mock-v3` management API. No live platform service is qualified. Credentials come from `SPOKE_TOKEN` or `token_file`. Organization-only configuration can bootstrap projects and environments; project/environment assertions narrow access for dependent services.",
		Attributes: map[string]schema.Attribute{
			"endpoint":        schema.StringAttribute{Required: true, MarkdownDescription: "Explicit HTTPS management origin. No default production target."},
			"organization_id": schema.StringAttribute{Required: true, Validators: idValidators(), MarkdownDescription: "Expected organization ID."},
			"project_id":      schema.StringAttribute{Optional: true, Validators: idValidators(), MarkdownDescription: "Optional target assertion. Omit for organization-scoped bootstrap; a configured value must already be known."},
			"environment_id":  schema.StringAttribute{Optional: true, Validators: idValidators(), MarkdownDescription: "Optional target assertion requiring project_id. Required for bucket resources; omit when managing parent scopes."},
			"token_file":      schema.StringAttribute{Optional: true, MarkdownDescription: "Path to an external machine credential. Takes precedence over `SPOKE_TOKEN`; credential bytes are not schema attributes."},
			"ca_file":         schema.StringAttribute{Optional: true, MarkdownDescription: "Path to additional PEM CA certificates. TLS verification remains enabled."},
			"timeout_seconds": schema.Int64Attribute{Optional: true, MarkdownDescription: "Total timeout per lifecycle operation, including polling; defaults to 60 seconds. Range: 1–3600."},
		},
	}
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open configured file")
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return nil, errors.New("cannot read configured file or file exceeds 1 MiB")
	}
	return b, nil
}

func configure(m providerModel) (*configured, error) {
	for _, v := range []types.String{m.Endpoint, m.OrganizationID, m.ProjectID, m.EnvironmentID, m.TokenFile, m.CAFile} {
		if v.IsUnknown() {
			return nil, errors.New("provider configuration must be known before accessing the management API")
		}
	}
	if m.TimeoutSeconds.IsUnknown() {
		return nil, errors.New("timeout_seconds must be known")
	}
	seconds := int64(60)
	if !m.TimeoutSeconds.IsNull() {
		seconds = m.TimeoutSeconds.ValueInt64()
	}
	if seconds < 1 || seconds > 3600 {
		return nil, errors.New("timeout_seconds must be between 1 and 3600")
	}
	token := strings.TrimSpace(os.Getenv("SPOKE_TOKEN"))
	if !m.TokenFile.IsNull() {
		b, err := readFile(m.TokenFile.ValueString())
		if err != nil {
			return nil, errors.New("cannot read token_file")
		}
		token = strings.TrimSpace(string(b))
	}
	var ca []byte
	if !m.CAFile.IsNull() {
		var err error
		ca, err = readFile(m.CAFile.ValueString())
		if err != nil {
			return nil, errors.New("cannot read ca_file")
		}
	}
	c, err := platformclient.New(platformclient.Config{
		Endpoint: m.Endpoint.ValueString(), Token: token, CAPEM: ca,
		Scope: platformclient.Scope{OrganizationID: m.OrganizationID.ValueString(), ProjectID: m.ProjectID.ValueString(), EnvironmentID: m.EnvironmentID.ValueString()},
	})
	if err != nil {
		return nil, err
	}
	return &configured{client: c, timeout: time.Duration(seconds) * time.Second}, nil
}

func (p *spokeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var m providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := configure(m)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Spoke provider configuration", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *spokeProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newBucketResource, newProjectResource, newEnvironmentResource, newServiceAccountResource, newGrantResource, func() resource.Resource { return &scopeResource{kind: "secret_reference"} }, func() resource.Resource { return &scopeResource{kind: "service_binding"} }, func() resource.Resource { return &scopeResource{kind: "database"} }, func() resource.Resource { return &scopeResource{kind: "artifact_registry"} }}
}
func (p *spokeProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newProjectDataSource, newEnvironmentDataSource, newProjectsDataSource, newEnvironmentsDataSource, newBucketsDataSource, newServiceAccountsDataSource, newGrantsDataSource, newArtifactDataSource, func() datasource.DataSource { return &serviceDataSource{kind: "secret_reference"} }, func() datasource.DataSource { return &collectionDataSource{kind: "secret_reference"} }, func() datasource.DataSource { return &serviceDataSource{kind: "service_binding"} }, func() datasource.DataSource { return &collectionDataSource{kind: "service_binding"} }, func() datasource.DataSource { return &serviceDataSource{kind: "database"} }, func() datasource.DataSource { return &collectionDataSource{kind: "database"} }, func() datasource.DataSource { return &serviceDataSource{kind: "bucket"} }, func() datasource.DataSource { return &serviceDataSource{kind: "artifact_registry"} }, func() datasource.DataSource { return &collectionDataSource{kind: "artifact_registry"} }}
}
