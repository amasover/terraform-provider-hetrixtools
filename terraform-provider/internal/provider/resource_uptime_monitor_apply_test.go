package provider

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	hetrixtools "github.com/xaf/terraform-provider-hetrixtools/client"
)

// A plan leaves unknown every optional, computed attribute the configuration does not
// set. Create and Update save the plan as the new state, so the state they return must
// still be fully known: Terraform refuses an unknown value after apply.
func TestUptimeMonitorApplyLeavesNoUnknownValues(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/test-token/uptime/add/" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"SUCCESS","monitor_id":"mid-1","server_id":"srv-1"}`))
	}))
	defer server.Close()
	client := hetrixtools.NewClientWithBaseURL(server.URL+"/v3", "test-token")

	// What a configuration sets, where the monitor type has the attribute; the rest is unknown.
	configured := map[string]tftypes.Value{
		"name":      tftypes.NewValue(tftypes.String, "Homepage"),
		"target":    tftypes.NewValue(tftypes.String, "https://example.com/"),
		"port":      tftypes.NewValue(tftypes.Number, big.NewFloat(25)),
		"locations": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "amsterdam")}),
	}

	for _, monitorType := range []uptimeMonitorType{uptimeMonitorHTTP, uptimeMonitorPing, uptimeMonitorSMTP, uptimeMonitorHeartbeat} {
		t.Run(string(monitorType), func(t *testing.T) {
			ctx := context.Background()
			r := &uptimeMonitorResource{client: client, monitorType: monitorType}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			plan := func(id tftypes.Value) tfsdk.Plan {
				values := map[string]tftypes.Value{}
				for name, attributeType := range objectType.AttributeTypes {
					values[name] = tftypes.NewValue(attributeType, tftypes.UnknownValue)
					if value, ok := configured[name]; ok {
						values[name] = value
					}
				}
				values["id"] = id
				return tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, values)}
			}

			created := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, nil)}}
			r.Create(ctx, resource.CreateRequest{Plan: plan(tftypes.NewValue(tftypes.String, tftypes.UnknownValue))}, &created)
			requireFullyKnown(t, "Create", created.Diagnostics, created.State)
			var id types.String
			created.State.GetAttribute(ctx, path.Root("id"), &id)
			if id.ValueString() != "mid-1" {
				t.Fatalf("Create saved id %s, want mid-1", id)
			}

			updated := resource.UpdateResponse{State: created.State}
			r.Update(ctx, resource.UpdateRequest{Plan: plan(tftypes.NewValue(tftypes.String, "mid-1")), State: created.State}, &updated)
			requireFullyKnown(t, "Update", updated.Diagnostics, updated.State)
		})
	}
}

func requireFullyKnown(t *testing.T, step string, diagnostics diag.Diagnostics, state tfsdk.State) {
	t.Helper()
	if diagnostics.HasError() {
		t.Fatalf("%s: %v", step, diagnostics)
	}
	if !state.Raw.IsFullyKnown() {
		t.Fatalf("%s saved unknown values: %s", step, state.Raw)
	}
}
