package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/client"
)

var (
	_ resource.Resource              = &SnapshotResource{}
	_ resource.ResourceWithConfigure = &SnapshotResource{}
)

func NewSnapshotResource() resource.Resource {
	return &SnapshotResource{}
}

type SnapshotResource struct {
	auraApi *client.AuraApi
}

type SnapshotResourceModel struct {
	InstanceId types.String `tfsdk:"instance_id"`
	SnapshotId types.String `tfsdk:"snapshot_id"`
	Profile    types.String `tfsdk:"profile"`
	Status     types.String `tfsdk:"status"`
	Timestamp  types.String `tfsdk:"timestamp"`
}

func (r *SnapshotResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	auraClient, ok := request.ProviderData.(*client.AuraClient)

	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AuraClient, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}
	r.auraApi = client.NewAuraApi(auraClient)
}

func (r *SnapshotResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_snapshot"
}

func (r *SnapshotResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Resource for an instance snapshot",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Id of the instance",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"snapshot_id": schema.StringAttribute{
				MarkdownDescription: "Id of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: "Profile of the snapshot. One of [AddHoc, Scheduled]",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of the snapshot. One of [Completed, InProgress, Failed, Pending]",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timestamp": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SnapshotResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data SnapshotResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	postResponse, err := r.auraApi.PostSnapshot(data.InstanceId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while creating a snapshot", err.Error())
		return
	}

	snapshot, err := r.auraApi.WaitUntilSnapshotIsInState(ctx, data.InstanceId.ValueString(), postResponse.Data.SnapshotId,
		func(resp client.GetSnapshotData) bool {
			return strings.ToLower(resp.Status) == "completed"
		})
	if err != nil {
		response.Diagnostics.AddError("Error while waiting for a snapshot", err.Error())
		return
	}

	data.SnapshotId = types.StringValue(snapshot.SnapshotId)
	data.Timestamp = types.StringValue(snapshot.Timestamp)
	data.Status = types.StringValue(snapshot.Status)
	data.Profile = types.StringValue(snapshot.Profile)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data SnapshotResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *SnapshotResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	// no op
}

func (r *SnapshotResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	// snapshots are immutable
}
