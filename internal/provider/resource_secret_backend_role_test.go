package provider

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func setOf(t *testing.T, vals ...string) types.Set {
	t.Helper()
	s, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return s
}

func TestRoleDataOmitsUnsetTTLs(t *testing.T) {
	m := secretBackendRoleModel{
		Apps:   setOf(t, "sonarr", "radarr", "prowlarr"),
		TTL:    types.Int64Null(),
		MaxTTL: types.Int64Null(),
	}
	data, diags := roleData(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("roleData: %v", diags)
	}
	apps, ok := data["apps"].([]string)
	if !ok || len(apps) != 3 || apps[0] != "sonarr" {
		t.Errorf("apps = %v, want [sonarr radarr prowlarr]", data["apps"])
	}
	if _, ok := data["ttl"]; ok {
		t.Errorf("ttl should be omitted when null")
	}
	if _, ok := data["max_ttl"]; ok {
		t.Errorf("max_ttl should be omitted when null")
	}
}

func TestRoleDataSendsEmptyMethodsWhenNull(t *testing.T) {
	// The engine only clears a method scope when the key is present, so an
	// unset methods must still be written as an empty list.
	m := secretBackendRoleModel{
		Apps:    setOf(t, "sonarr"),
		Methods: types.SetNull(types.StringType),
		TTL:     types.Int64Null(),
		MaxTTL:  types.Int64Null(),
	}
	data, diags := roleData(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("roleData: %v", diags)
	}
	methods, ok := data["methods"].([]string)
	if !ok {
		t.Fatalf("methods = %#v, want an empty []string", data["methods"])
	}
	if len(methods) != 0 {
		t.Errorf("methods = %v, want empty", methods)
	}
}

func TestRoleDataIncludesMethods(t *testing.T) {
	m := secretBackendRoleModel{
		Apps:    setOf(t, "sonarr"),
		Methods: setOf(t, "GET", "HEAD"),
		TTL:     types.Int64Null(),
		MaxTTL:  types.Int64Null(),
	}
	data, diags := roleData(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("roleData: %v", diags)
	}
	methods, ok := data["methods"].([]string)
	if !ok || len(methods) != 2 {
		t.Fatalf("methods = %#v, want [GET HEAD]", data["methods"])
	}
	sort.Strings(methods)
	if methods[0] != "GET" || methods[1] != "HEAD" {
		t.Errorf("methods = %v, want [GET HEAD]", methods)
	}
}

func TestRoleDataIncludesTTLs(t *testing.T) {
	m := secretBackendRoleModel{
		Apps:   setOf(t, "prowlarr"),
		TTL:    types.Int64Value(60),
		MaxTTL: types.Int64Value(86400),
	}
	data, diags := roleData(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("roleData: %v", diags)
	}
	if data["ttl"] != int64(60) {
		t.Errorf("ttl = %v, want 60", data["ttl"])
	}
	if data["max_ttl"] != int64(86400) {
		t.Errorf("max_ttl = %v, want 86400", data["max_ttl"])
	}
}

func TestApplyRoleDataMapsEngineResponse(t *testing.T) {
	// Shape mirrors what the engine's role read returns via the Vault API.
	role := map[string]interface{}{
		"apps":    []interface{}{"sonarr", "radarr", "prowlarr"},
		"ttl":     json.Number("60"),
		"max_ttl": json.Number("86400"),
	}
	var m secretBackendRoleModel
	m.TTL = types.Int64Null()
	m.MaxTTL = types.Int64Null()
	if diags := applyRoleData(&m, role); diags.HasError() {
		t.Fatalf("applyRoleData: %v", diags)
	}
	if !m.Apps.Equal(setOf(t, "sonarr", "radarr", "prowlarr")) {
		t.Errorf("apps = %v, want {sonarr radarr prowlarr}", m.Apps)
	}
	if m.TTL.ValueInt64() != 60 || m.MaxTTL.ValueInt64() != 86400 {
		t.Errorf("ttl/max_ttl = %d/%d, want 60/86400", m.TTL.ValueInt64(), m.MaxTTL.ValueInt64())
	}
}

func TestApplyRoleDataOrderInsensitive(t *testing.T) {
	// The engine returns apps alphabetically sorted regardless of the order
	// they were written in; the read-back must still equal the config value.
	configApps := setOf(t, "sonarr", "prowlarr", "radarr")
	role := map[string]interface{}{
		"apps": []interface{}{"prowlarr", "radarr", "sonarr"},
	}
	m := secretBackendRoleModel{
		Apps:   configApps,
		TTL:    types.Int64Null(),
		MaxTTL: types.Int64Null(),
	}
	if diags := applyRoleData(&m, role); diags.HasError() {
		t.Fatalf("applyRoleData: %v", diags)
	}
	if !m.Apps.Equal(configApps) {
		t.Errorf("apps after read-back = %v, not equal to config value %v", m.Apps, configApps)
	}
}

func TestApplyRoleDataZeroTTLLeavesNull(t *testing.T) {
	// The engine returns 0 for an unset TTL, which must not clobber the null
	// model value into a spurious 0.
	role := map[string]interface{}{
		"apps":    []interface{}{"sonarr"},
		"ttl":     json.Number("0"),
		"max_ttl": json.Number("0"),
	}
	m := secretBackendRoleModel{
		TTL:    types.Int64Null(),
		MaxTTL: types.Int64Null(),
	}
	if diags := applyRoleData(&m, role); diags.HasError() {
		t.Fatalf("applyRoleData: %v", diags)
	}
	if !m.TTL.IsNull() {
		t.Errorf("ttl = %v, want null", m.TTL)
	}
	if !m.MaxTTL.IsNull() {
		t.Errorf("max_ttl = %v, want null", m.MaxTTL)
	}
}

func TestApplyRoleDataMapsMethods(t *testing.T) {
	// The engine upper-cases and sorts the method scope it stores.
	role := map[string]interface{}{
		"apps":    []interface{}{"sonarr"},
		"methods": []interface{}{"GET", "HEAD"},
	}
	m := secretBackendRoleModel{
		Methods: setOf(t, "HEAD", "GET"),
		TTL:     types.Int64Null(),
		MaxTTL:  types.Int64Null(),
	}
	if diags := applyRoleData(&m, role); diags.HasError() {
		t.Fatalf("applyRoleData: %v", diags)
	}
	if !m.Methods.Equal(setOf(t, "GET", "HEAD")) {
		t.Errorf("methods = %v, want {GET HEAD}", m.Methods)
	}
}

func TestApplyRoleDataUnrestrictedMethodsStayNull(t *testing.T) {
	// An unrestricted role reads back as an empty/absent list, which must not
	// clobber a null config value into an empty set and show permanent drift.
	for name, role := range map[string]map[string]interface{}{
		"absent": {"apps": []interface{}{"sonarr"}},
		"empty":  {"apps": []interface{}{"sonarr"}, "methods": []interface{}{}},
		"null":   {"apps": []interface{}{"sonarr"}, "methods": nil},
	} {
		t.Run(name, func(t *testing.T) {
			m := secretBackendRoleModel{
				Methods: types.SetNull(types.StringType),
				TTL:     types.Int64Null(),
				MaxTTL:  types.Int64Null(),
			}
			if diags := applyRoleData(&m, role); diags.HasError() {
				t.Fatalf("applyRoleData: %v", diags)
			}
			if !m.Methods.IsNull() {
				t.Errorf("methods = %v, want null", m.Methods)
			}
		})
	}
}

func TestApplyRoleDataClearedMethodsShowDrift(t *testing.T) {
	// A scope removed out of band must land in state as empty, so the next plan
	// proposes putting the configured methods back.
	role := map[string]interface{}{"apps": []interface{}{"sonarr"}}
	m := secretBackendRoleModel{
		Methods: setOf(t, "GET"),
		TTL:     types.Int64Null(),
		MaxTTL:  types.Int64Null(),
	}
	if diags := applyRoleData(&m, role); diags.HasError() {
		t.Fatalf("applyRoleData: %v", diags)
	}
	if m.Methods.IsNull() || len(m.Methods.Elements()) != 0 {
		t.Errorf("methods = %v, want an empty set", m.Methods)
	}
}

func TestRoleSchemaMethodsValidation(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	NewSecretBackendRoleResource().(*secretBackendRoleResource).Schema(ctx, resource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["methods"].(schema.SetAttribute)
	if !ok {
		t.Fatalf("methods attribute = %#v, want a schema.SetAttribute", resp.Schema.Attributes["methods"])
	}
	if attr.IsRequired() || !attr.IsOptional() {
		t.Errorf("methods should be optional, not required")
	}
	if len(attr.Validators) == 0 {
		t.Fatal("methods should carry a value validator")
	}

	for _, tc := range []struct {
		name    string
		values  []string
		wantErr bool
	}{
		{name: "allowed", values: []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}},
		{name: "unknown method", values: []string{"GET", "FETCH"}, wantErr: true},
		{name: "lower case", values: []string{"get"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.SetRequest{
				Path:        path.Root("methods"),
				ConfigValue: setOf(t, tc.values...),
			}
			var vResp validator.SetResponse
			for _, v := range attr.Validators {
				v.ValidateSet(ctx, req, &vResp)
			}
			if got := vResp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("validation error = %v, want %v (%v)", got, tc.wantErr, vResp.Diagnostics)
			}
		})
	}
}
