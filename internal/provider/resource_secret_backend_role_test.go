package provider

import (
	"context"
	"encoding/json"
	"testing"

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
