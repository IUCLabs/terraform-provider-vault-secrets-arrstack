package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &arrstackProvider{}

type arrstackProvider struct {
	version string
}

type arrstackProviderModel struct {
	Address types.String `tfsdk:"address"`
	Token   types.String `tfsdk:"token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &arrstackProvider{version: version}
	}
}

func (p *arrstackProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	// Registry type is vault-secrets-arrstack; declare it under the local name "arrstack".
	resp.TypeName = "arrstack"
	resp.Version = p.version
}

func (p *arrstackProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages the arrstack secrets engine (vault-plugin-secrets-arrstack) on HashiCorp Vault or OpenBao: its mount, arrproxy connection config, and roles.",
		MarkdownDescription: "Manages the arrstack secrets engine ([vault-plugin-secrets-arrstack](https://github.com/IUCLabs/vault-plugin-secrets-arrstack)) on HashiCorp Vault or OpenBao: its mount, arrproxy connection config, and roles. The engine mints short-lived arrproxy machine tokens scoped to Sonarr, Radarr, and Prowlarr.",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				Description: "Address of the Vault/OpenBao server. Falls back to the VAULT_ADDR environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Token used to authenticate to Vault/OpenBao. Falls back to the VAULT_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *arrstackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config arrstackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	address := os.Getenv("VAULT_ADDR")
	if !config.Address.IsNull() && config.Address.ValueString() != "" {
		address = config.Address.ValueString()
	}

	token := os.Getenv("VAULT_TOKEN")
	if !config.Token.IsNull() && config.Token.ValueString() != "" {
		token = config.Token.ValueString()
	}

	if address == "" {
		resp.Diagnostics.AddError(
			"missing Vault address",
			"Set the provider \"address\" attribute or the VAULT_ADDR environment variable.",
		)
		return
	}

	client, err := newVaultClient(address, token)
	if err != nil {
		resp.Diagnostics.AddError("failed to create Vault client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *arrstackProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSecretBackendResource,
		NewSecretBackendRoleResource,
	}
}

func (p *arrstackProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
