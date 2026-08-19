package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func listOf(t *testing.T, vals ...string) types.List {
	t.Helper()
	l, diags := types.ListValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		t.Fatalf("building list: %v", diags)
	}
	return l
}

func TestRoleDataOmitsUnsetTTLs(t *testing.T) {
	m := secretBackendRoleModel{
		Apps:   listOf(t, "sonarr", "radarr", "prowlarr"),
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

func TestRoleDataIncludesTTLs(t *testing.T) {
	m := secretBackendRoleModel{
		Apps:   listOf(t, "prowlarr"),
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
	var apps []string
	m.Apps.ElementsAs(context.Background(), &apps, false)
	if len(apps) != 3 || apps[2] != "prowlarr" {
		t.Errorf("apps = %v, want [sonarr radarr prowlarr]", apps)
	}
	if m.TTL.ValueInt64() != 60 || m.MaxTTL.ValueInt64() != 86400 {
		t.Errorf("ttl/max_ttl = %d/%d, want 60/86400", m.TTL.ValueInt64(), m.MaxTTL.ValueInt64())
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
