package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &secretBackendRoleResource{}
	_ resource.ResourceWithImportState = &secretBackendRoleResource{}
)

type secretBackendRoleResource struct {
	client *vaultClient
}

type secretBackendRoleModel struct {
	Backend types.String `tfsdk:"backend"`
	Name    types.String `tfsdk:"name"`
	Apps    types.List   `tfsdk:"apps"`
	TTL     types.Int64  `tfsdk:"ttl"`
	MaxTTL  types.Int64  `tfsdk:"max_ttl"`
}

func NewSecretBackendRoleResource() resource.Resource {
	return &secretBackendRoleResource{}
}

func (r *secretBackendRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_backend_role"
}

func (r *secretBackendRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a role on the arrstack secrets engine that mints short-lived scoped arrproxy API keys.",
		Attributes: map[string]schema.Attribute{
			"backend": schema.StringAttribute{
				Description: "Mount path of the arrstack secrets engine.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"apps": schema.ListAttribute{
				Description: "arr apps a generated key may access (subset of sonarr, radarr, prowlarr).",
				ElementType: types.StringType,
				Required:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "Default lease TTL in seconds for keys generated from this role.",
				Optional:    true,
			},
			"max_ttl": schema.Int64Attribute{
				Description: "Maximum lease TTL in seconds for keys generated from this role.",
				Optional:    true,
			},
		},
	}
}

func (r *secretBackendRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*vaultClient)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *secretBackendRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretBackendRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data, diags := roleData(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.write(ctx, rolePath(plan.Backend.ValueString(), plan.Name.ValueString()), data); err != nil {
		resp.Diagnostics.AddError("failed to create arrstack role", err.Error())
		return
	}
	resp.Diagnostics.Append(r.readInto(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *secretBackendRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretBackendRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	role, err := r.client.read(ctx, rolePath(state.Backend.ValueString(), state.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("failed to read arrstack role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(applyRoleData(&state, role)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *secretBackendRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secretBackendRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data, diags := roleData(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.write(ctx, rolePath(plan.Backend.ValueString(), plan.Name.ValueString()), data); err != nil {
		resp.Diagnostics.AddError("failed to update arrstack role", err.Error())
		return
	}
	resp.Diagnostics.Append(r.readInto(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *secretBackendRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretBackendRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.delete(ctx, rolePath(state.Backend.ValueString(), state.Name.ValueString())); err != nil {
		resp.Diagnostics.AddError("failed to delete arrstack role", err.Error())
		return
	}
}

func (r *secretBackendRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	backend, name, ok := splitBackendName(req.ID, "roles")
	if !ok {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf("expected \"<backend>/roles/<name>\", got %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("backend"), backend)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

func (r *secretBackendRoleResource) readInto(ctx context.Context, m *secretBackendRoleModel) diag.Diagnostics {
	var diags diag.Diagnostics
	role, err := r.client.read(ctx, rolePath(m.Backend.ValueString(), m.Name.ValueString()))
	if err != nil {
		diags.AddError("failed to read back arrstack role", err.Error())
		return diags
	}
	if role == nil {
		diags.AddError("role missing after write", "the role was not found immediately after being written")
		return diags
	}
	return applyRoleData(m, role)
}

func roleData(ctx context.Context, m secretBackendRoleModel) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	data := map[string]interface{}{}

	if !m.Apps.IsNull() && !m.Apps.IsUnknown() {
		var apps []string
		diags.Append(m.Apps.ElementsAs(ctx, &apps, false)...)
		data["apps"] = apps
	}
	if !m.TTL.IsNull() && !m.TTL.IsUnknown() {
		data["ttl"] = m.TTL.ValueInt64()
	}
	if !m.MaxTTL.IsNull() && !m.MaxTTL.IsUnknown() {
		data["max_ttl"] = m.MaxTTL.ValueInt64()
	}
	return data, diags
}

func applyRoleData(m *secretBackendRoleModel, role map[string]interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	apps := toStringSlice(role["apps"])
	appList, appDiags := types.ListValueFrom(context.Background(), types.StringType, apps)
	diags.Append(appDiags...)
	m.Apps = appList

	if n, ok := toInt64(role["ttl"]); ok && n != 0 {
		m.TTL = types.Int64Value(n)
	} else if m.TTL.IsUnknown() {
		m.TTL = types.Int64Null()
	}
	if n, ok := toInt64(role["max_ttl"]); ok && n != 0 {
		m.MaxTTL = types.Int64Value(n)
	} else if m.MaxTTL.IsUnknown() {
		m.MaxTTL = types.Int64Null()
	}
	return diags
}
