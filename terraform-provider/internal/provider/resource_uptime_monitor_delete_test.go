package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	hetrixtools "github.com/xaf/terraform-provider-hetrixtools/client"
)

// Delete must find the monitor's id in the state of every monitor type, each of which
// holds attributes of its own beside the common ones.
func TestUptimeMonitorDeleteReadsTheIDOfEveryType(t *testing.T) {
	t.Parallel()

	for _, monitorType := range []uptimeMonitorType{uptimeMonitorHTTP, uptimeMonitorPing, uptimeMonitorSMTP, uptimeMonitorHeartbeat} {
		t.Run(string(monitorType), func(t *testing.T) {
			t.Parallel()

			deleted := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v2/test-token/uptime/delete/" {
					t.Errorf("unexpected path %q", r.URL.Path)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode delete body: %s", err)
				}
				deleted <- body["MID"]
				_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
			}))
			defer server.Close()

			ctx := context.Background()
			r := &uptimeMonitorResource{client: hetrixtools.NewClientWithBaseURL(server.URL+"/v3", "test-token"), monitorType: monitorType}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			objectType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
			values := map[string]tftypes.Value{}
			for name, attributeType := range objectType.AttributeTypes {
				values[name] = tftypes.NewValue(attributeType, nil)
			}
			values["id"] = tftypes.NewValue(tftypes.String, "mid-1")
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, values)}

			resp := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Delete: %v", resp.Diagnostics)
			}
			select {
			case id := <-deleted:
				if id != "mid-1" {
					t.Fatalf("deleted MID %q, want mid-1", id)
				}
			default:
				t.Fatal("Delete sent no delete request")
			}
		})
	}
}
