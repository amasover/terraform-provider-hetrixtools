package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	hetrixtools "github.com/xaf/terraform-provider-hetrixtools/client"
)

var _ resource.Resource = (*uptimeMonitorResource)(nil)
var _ resource.ResourceWithConfigure = (*uptimeMonitorResource)(nil)
var _ resource.ResourceWithImportState = (*uptimeMonitorResource)(nil)

type uptimeMonitorType string

const (
	uptimeMonitorHTTP      uptimeMonitorType = "http"
	uptimeMonitorPing      uptimeMonitorType = "ping"
	uptimeMonitorSMTP      uptimeMonitorType = "smtp"
	uptimeMonitorHeartbeat uptimeMonitorType = "heartbeat"
)

type uptimeMonitorResource struct {
	client      *hetrixtools.Client
	monitorType uptimeMonitorType
}

type uptimeCommonModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Timeout          types.Int64  `tfsdk:"timeout"`
	Frequency        types.Int64  `tfsdk:"frequency"`
	FailsBeforeAlert types.Int64  `tfsdk:"fails_before_alert"`
	ContactList      types.String `tfsdk:"contact_list_id"`
	Category         types.String `tfsdk:"category"`
	AlertAfter       types.String `tfsdk:"alert_after"`
	RepeatTimes      types.Int64  `tfsdk:"repeat_times"`
	RepeatEvery      types.String `tfsdk:"repeat_every"`
	Public           types.Bool   `tfsdk:"public"`
	ShowTarget       types.Bool   `tfsdk:"show_target"`
}

type uptimeHTTPMonitorModel struct {
	uptimeCommonModel
	Target          types.String `tfsdk:"target"`
	FailedLocations types.Int64  `tfsdk:"failed_locations"`
	Locations       types.Set    `tfsdk:"locations"`
	HTTPMethod      types.String `tfsdk:"http_method"`
	MaxRedirects    types.Int64  `tfsdk:"max_redirects"`
	Keyword         types.String `tfsdk:"keyword"`
	HTTPCodes       types.List   `tfsdk:"accepted_http_codes"`
	VerSSLCert      types.Bool   `tfsdk:"verify_ssl_certificate"`
	VerSSLHost      types.Bool   `tfsdk:"verify_ssl_host"`
}

type uptimePingMonitorModel struct {
	uptimeCommonModel
	Target          types.String `tfsdk:"target"`
	FailedLocations types.Int64  `tfsdk:"failed_locations"`
	Locations       types.Set    `tfsdk:"locations"`
}

type uptimeSMTPMonitorModel struct {
	uptimeCommonModel
	Target          types.String `tfsdk:"target"`
	Port            types.Int64  `tfsdk:"port"`
	SMTPUser        types.String `tfsdk:"smtp_user"`
	SMTPPass        types.String `tfsdk:"smtp_password"`
	FailedLocations types.Int64  `tfsdk:"failed_locations"`
	Locations       types.Set    `tfsdk:"locations"`
	VerSSLCert      types.Bool   `tfsdk:"verify_ssl_certificate"`
	VerSSLHost      types.Bool   `tfsdk:"verify_ssl_host"`
}

type uptimeHeartbeatMonitorModel struct {
	uptimeCommonModel
	Grace      types.Int64  `tfsdk:"grace"`
	InfoPublic types.Bool   `tfsdk:"info_public"`
	CPUPublic  types.Bool   `tfsdk:"cpu_public"`
	RAMPublic  types.Bool   `tfsdk:"ram_public"`
	DiskPublic types.Bool   `tfsdk:"disk_public"`
	NetPublic  types.Bool   `tfsdk:"net_public"`
	ServerID   types.String `tfsdk:"server_id"`
}

func newUptimeHTTPMonitorResource() resource.Resource {
	return &uptimeMonitorResource{monitorType: uptimeMonitorHTTP}
}

func newUptimePingMonitorResource() resource.Resource {
	return &uptimeMonitorResource{monitorType: uptimeMonitorPing}
}

func newUptimeSMTPMonitorResource() resource.Resource {
	return &uptimeMonitorResource{monitorType: uptimeMonitorSMTP}
}

func newUptimeHeartbeatMonitorResource() resource.Resource {
	return &uptimeMonitorResource{monitorType: uptimeMonitorHeartbeat}
}

func (r *uptimeMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitor_" + string(r.monitorType)
}

func (r *uptimeMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := uptimeCommonAttributes()
	switch r.monitorType {
	case uptimeMonitorHTTP:
		attributes["target"] = schema.StringAttribute{Required: true, MarkdownDescription: "URL to check."}
		addUptimeLocationAttributes(attributes)
		attributes["http_method"] = optionalComputedString()
		attributes["max_redirects"] = optionalComputedInt64()
		attributes["keyword"] = optionalComputedString()
		attributes["accepted_http_codes"] = optionalComputedInt64List()
		addUptimeSSLAttributes(attributes)
	case uptimeMonitorPing:
		attributes["target"] = schema.StringAttribute{Required: true, MarkdownDescription: "Hostname or IP address to ping."}
		addUptimeLocationAttributes(attributes)
	case uptimeMonitorSMTP:
		attributes["target"] = schema.StringAttribute{Required: true, MarkdownDescription: "SMTP hostname."}
		attributes["port"] = schema.Int64Attribute{Required: true, MarkdownDescription: "SMTP port."}
		attributes["smtp_user"] = optionalComputedString()
		attributes["smtp_password"] = optionalComputedSensitiveString()
		addUptimeLocationAttributes(attributes)
		addUptimeSSLAttributes(attributes)
	case uptimeMonitorHeartbeat:
		attributes["grace"] = optionalComputedInt64()
		attributes["info_public"] = optionalComputedBool()
		attributes["cpu_public"] = optionalComputedBool()
		attributes["ram_public"] = optionalComputedBool()
		attributes["disk_public"] = optionalComputedBool()
		attributes["net_public"] = optionalComputedBool()
		attributes["server_id"] = schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("Manages a HetrixTools %s uptime monitor.", r.monitorType),
		Attributes:          attributes,
	}
}

func uptimeCommonAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name":               schema.StringAttribute{Required: true},
		"timeout":            optionalComputedInt64(),
		"frequency":          optionalComputedInt64(),
		"fails_before_alert": optionalComputedInt64(),
		"contact_list_id":    optionalComputedString(),
		"category":           optionalComputedString(),
		"alert_after":        optionalComputedString(),
		"repeat_times":       optionalComputedInt64(),
		"repeat_every":       optionalComputedString(),
		"public":             optionalComputedBool(),
		"show_target":        optionalComputedBool(),
	}
}

func addUptimeLocationAttributes(attributes map[string]schema.Attribute) {
	attributes["failed_locations"] = optionalComputedInt64()
	attributes["locations"] = schema.SetAttribute{
		Optional:            true,
		Computed:            true,
		ElementType:         types.StringType,
		MarkdownDescription: "Set of canonical HetrixTools v3 location names enabled for this monitor, e.g. `[\"amsterdam\", \"new_york\"]`.",
		PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
	}
}

func addUptimeSSLAttributes(attributes map[string]schema.Attribute) {
	attributes["verify_ssl_certificate"] = optionalComputedBool()
	attributes["verify_ssl_host"] = optionalComputedBool()
}

func optionalComputedString() schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
}

func optionalComputedSensitiveString() schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
}

func optionalComputedInt64() schema.Int64Attribute {
	return schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}}
}

func optionalComputedBool() schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
}

func optionalComputedInt64List() schema.ListAttribute {
	return schema.ListAttribute{Optional: true, Computed: true, ElementType: types.Int64Type, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}}
}

func (r *uptimeMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*hetrixtools.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *hetrixtools.Client.")
		return
	}
	r.client = c
}

func (r *uptimeMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	switch r.monitorType {
	case uptimeMonitorHTTP:
		var plan uptimeHTTPMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.CreateUptimeMonitor(ctx, uptimeHTTPMonitorRequestFromModel(ctx, plan, &resp.Diagnostics))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, nil) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	case uptimeMonitorPing:
		var plan uptimePingMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.CreateUptimeMonitor(ctx, uptimePingMonitorRequestFromModel(ctx, plan, &resp.Diagnostics))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, nil) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	case uptimeMonitorSMTP:
		var plan uptimeSMTPMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.CreateUptimeMonitor(ctx, uptimeSMTPMonitorRequestFromModel(ctx, plan, &resp.Diagnostics))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, nil) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	case uptimeMonitorHeartbeat:
		var plan uptimeHeartbeatMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.CreateUptimeMonitor(ctx, uptimeHeartbeatMonitorRequestFromModel(plan))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, &plan.ServerID) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	}
}

func (r *uptimeMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	switch r.monitorType {
	case uptimeMonitorHTTP:
		var state uptimeHTTPMonitorModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor, ok := r.readUptimeMonitor(ctx, state.ID.ValueString(), &resp.Diagnostics, resp.State.RemoveResource)
		if !ok {
			return
		}
		state = uptimeHTTPMonitorModelFromAPI(ctx, state, *monitor, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	case uptimeMonitorPing:
		var state uptimePingMonitorModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor, ok := r.readUptimeMonitor(ctx, state.ID.ValueString(), &resp.Diagnostics, resp.State.RemoveResource)
		if !ok {
			return
		}
		state = uptimePingMonitorModelFromAPI(ctx, state, *monitor, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	case uptimeMonitorSMTP:
		var state uptimeSMTPMonitorModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor, ok := r.readUptimeMonitor(ctx, state.ID.ValueString(), &resp.Diagnostics, resp.State.RemoveResource)
		if !ok {
			return
		}
		state = uptimeSMTPMonitorModelFromAPI(ctx, state, *monitor, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	case uptimeMonitorHeartbeat:
		var state uptimeHeartbeatMonitorModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor, ok := r.readUptimeMonitor(ctx, state.ID.ValueString(), &resp.Diagnostics, resp.State.RemoveResource)
		if !ok {
			return
		}
		state = uptimeHeartbeatMonitorModelFromAPI(state, *monitor)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func (r *uptimeMonitorResource) readUptimeMonitor(ctx context.Context, id string, diagnostics interface{ AddError(string, string) }, remove func(context.Context)) (*hetrixtools.UptimeMonitor, bool) {
	found, err := r.client.GetUptimeMonitor(ctx, id)
	if err != nil {
		diagnostics.AddError("Read uptime monitor failed", err.Error())
		return nil, false
	}
	if found == nil {
		remove(ctx)
		return nil, false
	}
	if found.Type != string(r.monitorType) {
		diagnostics.AddError("Unexpected uptime monitor type", fmt.Sprintf("Imported monitor has type %q, but this resource manages %q monitors.", found.Type, r.monitorType))
		return nil, false
	}
	return found, true
}

func (r *uptimeMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	switch r.monitorType {
	case uptimeMonitorHTTP:
		var plan uptimeHTTPMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.UpdateUptimeMonitor(ctx, uptimeHTTPMonitorRequestFromModel(ctx, plan, &resp.Diagnostics))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, nil) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	case uptimeMonitorPing:
		var plan uptimePingMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.UpdateUptimeMonitor(ctx, uptimePingMonitorRequestFromModel(ctx, plan, &resp.Diagnostics))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, nil) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	case uptimeMonitorSMTP:
		var plan uptimeSMTPMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.UpdateUptimeMonitor(ctx, uptimeSMTPMonitorRequestFromModel(ctx, plan, &resp.Diagnostics))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, nil) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	case uptimeMonitorHeartbeat:
		var plan uptimeHeartbeatMonitorModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		action, err := r.client.UpdateUptimeMonitor(ctx, uptimeHeartbeatMonitorRequestFromModel(plan))
		if finishUptimeMutation(resp.Diagnostics.AddError, action, err, &plan.uptimeCommonModel, &plan.ServerID) {
			nullUnknowns(&plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		}
	}
}

func finishUptimeMutation(addError func(string, string), action *hetrixtools.ActionResponse, err error, common *uptimeCommonModel, serverID *types.String) bool {
	if err != nil {
		addError("Uptime monitor mutation failed", err.Error())
		return false
	}
	if action != nil {
		if action.MonitorID != "" {
			common.ID = types.StringValue(action.MonitorID)
		}
		if serverID != nil && action.ServerID != "" {
			*serverID = types.StringValue(action.ServerID)
		}
	}
	return true
}

func (r *uptimeMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state uptimeCommonModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.DeleteUptimeMonitor(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Delete uptime monitor failed", err.Error())
	}
}

func (r *uptimeMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRoot("id"), req, resp)
}

func uptimeCommonModelFromAPI(state uptimeCommonModel, monitor hetrixtools.UptimeMonitor) uptimeCommonModel {
	state.ID = types.StringValue(monitor.ID)
	state.Name = types.StringValue(monitor.Name)
	state.Timeout = types.Int64Value(monitor.Timeout)
	state.Frequency = types.Int64Value(monitor.Frequency)
	state.FailsBeforeAlert = types.Int64Value(monitor.FailsBeforeAlert)
	state.ContactList = stringNullIfEmpty(monitor.ContactListID)
	state.Category = stringNullIfEmpty(monitor.Category)
	state.AlertAfter = stringNullIfEmpty(monitor.AlertAfter)
	state.RepeatTimes = types.Int64Value(monitor.RepeatTimes)
	state.RepeatEvery = stringNullIfEmpty(monitor.RepeatEvery)
	state.Public = boolFromPointer(monitor.Public)
	state.ShowTarget = boolFromPointer(monitor.ShowTarget)
	return state
}

func uptimeHTTPMonitorModelFromAPI(ctx context.Context, state uptimeHTTPMonitorModel, monitor hetrixtools.UptimeMonitor, diagnostics interface{ AddError(string, string) }) uptimeHTTPMonitorModel {
	state.uptimeCommonModel = uptimeCommonModelFromAPI(state.uptimeCommonModel, monitor)
	state.Target = types.StringValue(monitor.Target)
	state.FailedLocations = types.Int64Value(monitor.FailedLocations)
	state.Locations = locationsSetFromAPI(ctx, state.Locations, monitor.Locations, diagnostics)
	state.HTTPMethod = stringNullIfEmpty(monitor.HTTPMethod)
	state.MaxRedirects = types.Int64Value(monitor.MaxRedirects)
	state.Keyword = stringNullIfEmpty(monitor.Keyword)
	state.HTTPCodes = int64ListFromAPI(ctx, state.HTTPCodes, monitor.HTTPCodes, diagnostics)
	state.VerSSLCert = boolFromPointer(monitor.VerSSLCert)
	state.VerSSLHost = boolFromPointer(monitor.VerSSLHost)
	return state
}

func uptimePingMonitorModelFromAPI(ctx context.Context, state uptimePingMonitorModel, monitor hetrixtools.UptimeMonitor, diagnostics interface{ AddError(string, string) }) uptimePingMonitorModel {
	state.uptimeCommonModel = uptimeCommonModelFromAPI(state.uptimeCommonModel, monitor)
	state.Target = types.StringValue(monitor.Target)
	state.FailedLocations = types.Int64Value(monitor.FailedLocations)
	state.Locations = locationsSetFromAPI(ctx, state.Locations, monitor.Locations, diagnostics)
	return state
}

func uptimeSMTPMonitorModelFromAPI(ctx context.Context, state uptimeSMTPMonitorModel, monitor hetrixtools.UptimeMonitor, diagnostics interface{ AddError(string, string) }) uptimeSMTPMonitorModel {
	state.uptimeCommonModel = uptimeCommonModelFromAPI(state.uptimeCommonModel, monitor)
	state.Target = types.StringValue(monitor.Target)
	if monitor.Port != nil {
		state.Port = types.Int64Value(*monitor.Port)
	}
	state.SMTPUser = stringNullIfEmpty(monitor.SMTPUser)
	if state.SMTPPass.IsNull() || state.SMTPPass.IsUnknown() {
		state.SMTPPass = types.StringNull()
	}
	state.FailedLocations = types.Int64Value(monitor.FailedLocations)
	state.Locations = locationsSetFromAPI(ctx, state.Locations, monitor.Locations, diagnostics)
	state.VerSSLCert = boolFromPointer(monitor.VerSSLCert)
	state.VerSSLHost = boolFromPointer(monitor.VerSSLHost)
	return state
}

func uptimeHeartbeatMonitorModelFromAPI(state uptimeHeartbeatMonitorModel, monitor hetrixtools.UptimeMonitor) uptimeHeartbeatMonitorModel {
	state.uptimeCommonModel = uptimeCommonModelFromAPI(state.uptimeCommonModel, monitor)
	state.Grace = types.Int64Value(monitor.Grace)
	state.InfoPublic = boolFromPointer(monitor.InfoPublic)
	state.CPUPublic = boolFromPointer(monitor.CPUPublic)
	state.RAMPublic = boolFromPointer(monitor.RAMPublic)
	state.DiskPublic = boolFromPointer(monitor.DiskPublic)
	state.NetPublic = boolFromPointer(monitor.NetPublic)
	if monitor.ServerID == nil {
		state.ServerID = types.StringNull()
	} else {
		state.ServerID = stringNullIfEmpty(*monitor.ServerID)
	}
	return state
}

func locationsSetFromAPI(ctx context.Context, current types.Set, locations []string, diagnostics interface{ AddError(string, string) }) types.Set {
	if locations == nil {
		return types.SetNull(types.StringType)
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, locations)
	if diags.HasError() {
		diagnostics.AddError("Invalid uptime monitor locations", fmt.Sprintf("Could not encode locations: %v", diags))
		return current
	}
	return set
}

func boolFromPointer(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func stringNullIfEmpty(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func int64ListFromAPI(ctx context.Context, current types.List, values []int64, diagnostics interface{ AddError(string, string) }) types.List {
	if len(values) == 0 {
		if current.IsNull() || current.IsUnknown() {
			return types.ListNull(types.Int64Type)
		}
		return current
	}
	list, diags := types.ListValueFrom(ctx, types.Int64Type, values)
	if diags.HasError() {
		diagnostics.AddError("Invalid uptime monitor integer list", fmt.Sprintf("Could not encode integer list: %v", diags))
		return current
	}
	return list
}

func uptimeCommonRequestFromModel(common uptimeCommonModel, monitorType uptimeMonitorType) hetrixtools.UptimeMonitorRequest {
	return hetrixtools.UptimeMonitorRequest{
		MID:              stringValue(common.ID, ""),
		Type:             string(monitorType),
		Name:             common.Name.ValueString(),
		Timeout:          int64Value(common.Timeout, 0),
		Frequency:        int64Value(common.Frequency, 0),
		FailsBeforeAlert: int64Value(common.FailsBeforeAlert, 0),
		ContactList:      stringValue(common.ContactList, ""),
		Category:         stringValue(common.Category, ""),
		AlertAfter:       stringValue(common.AlertAfter, ""),
		RepeatTimes:      int64Value(common.RepeatTimes, 0),
		RepeatEvery:      stringValue(common.RepeatEvery, ""),
		Public:           boolPointer(common.Public),
		ShowTarget:       boolPointer(common.ShowTarget),
	}
}

func uptimeHTTPMonitorRequestFromModel(ctx context.Context, model uptimeHTTPMonitorModel, diagnostics interface{ AddError(string, string) }) hetrixtools.UptimeMonitorRequest {
	request := uptimeCommonRequestFromModel(model.uptimeCommonModel, uptimeMonitorHTTP)
	request.Target = model.Target.ValueString()
	request.FailedLocations = int64Value(model.FailedLocations, 0)
	request.Locations = locationValuesFromSet(ctx, model.Locations, diagnostics)
	request.HTTPMethod = stringValue(model.HTTPMethod, "")
	request.MaxRedirects = int64Value(model.MaxRedirects, 0)
	request.Keyword = stringValue(model.Keyword, "")
	request.HTTPCodes = int64ValuesFromList(ctx, model.HTTPCodes, diagnostics)
	request.VerSSLCert = boolPointer(model.VerSSLCert)
	request.VerSSLHost = boolPointer(model.VerSSLHost)
	return request
}

func uptimePingMonitorRequestFromModel(ctx context.Context, model uptimePingMonitorModel, diagnostics interface{ AddError(string, string) }) hetrixtools.UptimeMonitorRequest {
	request := uptimeCommonRequestFromModel(model.uptimeCommonModel, uptimeMonitorPing)
	request.Target = model.Target.ValueString()
	request.FailedLocations = int64Value(model.FailedLocations, 0)
	request.Locations = locationValuesFromSet(ctx, model.Locations, diagnostics)
	return request
}

func uptimeSMTPMonitorRequestFromModel(ctx context.Context, model uptimeSMTPMonitorModel, diagnostics interface{ AddError(string, string) }) hetrixtools.UptimeMonitorRequest {
	request := uptimeCommonRequestFromModel(model.uptimeCommonModel, uptimeMonitorSMTP)
	request.Target = model.Target.ValueString()
	request.Port = model.Port.ValueInt64()
	request.SMTPUser = stringValue(model.SMTPUser, "")
	request.SMTPPass = stringValue(model.SMTPPass, "")
	request.FailedLocations = int64Value(model.FailedLocations, 0)
	request.Locations = locationValuesFromSet(ctx, model.Locations, diagnostics)
	request.VerSSLCert = boolPointer(model.VerSSLCert)
	request.VerSSLHost = boolPointer(model.VerSSLHost)
	return request
}

func uptimeHeartbeatMonitorRequestFromModel(model uptimeHeartbeatMonitorModel) hetrixtools.UptimeMonitorRequest {
	request := uptimeCommonRequestFromModel(model.uptimeCommonModel, uptimeMonitorHeartbeat)
	request.Grace = int64Value(model.Grace, 0)
	request.INFOPub = boolPointer(model.InfoPublic)
	request.CPUPub = boolPointer(model.CPUPublic)
	request.RAMPub = boolPointer(model.RAMPublic)
	request.DISKPub = boolPointer(model.DiskPublic)
	request.NETPub = boolPointer(model.NetPublic)
	return request
}

func locationValuesFromSet(ctx context.Context, locationsSet types.Set, diagnostics interface{ AddError(string, string) }) []string {
	if locationsSet.IsNull() || locationsSet.IsUnknown() {
		return nil
	}
	var locations []types.String
	if diags := locationsSet.ElementsAs(ctx, &locations, false); diags.HasError() {
		diagnostics.AddError("Invalid locations", fmt.Sprintf("Could not decode locations: %v", diags))
		return nil
	}
	values := make([]string, 0, len(locations))
	for _, location := range locations {
		if !location.IsNull() && !location.IsUnknown() {
			values = append(values, location.ValueString())
		}
	}
	return values
}

func int64ValuesFromList(ctx context.Context, list types.List, diagnostics interface{ AddError(string, string) }) []int64 {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var values []int64
	if diags := list.ElementsAs(ctx, &values, false); diags.HasError() {
		diagnostics.AddError("Invalid accepted_http_codes", fmt.Sprintf("Could not decode accepted_http_codes: %v", diags))
		return nil
	}
	return values
}

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}
