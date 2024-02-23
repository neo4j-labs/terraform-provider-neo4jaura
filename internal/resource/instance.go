package resource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/client"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/util"
	"strings"
)

// Ensure resource defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &InstanceResource{}
	_ resource.ResourceWithConfigure = &InstanceResource{}
)

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResource struct {
	auraApi *client.AuraApi
}

type InstanceResourceModel struct {
	InstanceId    types.String `tfsdk:"instance_id"`
	Name          types.String `tfsdk:"name"`
	Region        types.String `tfsdk:"region"`
	Memory        types.String `tfsdk:"memory"`
	Type          types.String `tfsdk:"type"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	TenantId      types.String `tfsdk:"tenant_id"`
	ConnectionUrl types.String `tfsdk:"connection_url"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	Version       types.String `tfsdk:"version"`
	Paused        types.Bool   `tfsdk:"paused"`
	Storage       types.String `tfsdk:"storage"`
	Source        types.Object `tfsdk:"source"`
}

type InstanceResourceSourceModel struct {
	InstanceId types.String `tfsdk:"instance_id"`
	SnapshotId types.String `tfsdk:"snapshot_id"`
}

func (r *InstanceResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *InstanceResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura instance",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Id of the instance",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name if the instance",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region of the instance",
				Required:            true,
			},
			"memory": schema.StringAttribute{
				MarkdownDescription: "Memory allocated for the instance",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("1GB"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("1GB", "2GB", "4GB", "8GB", "16GB", "24GB", "32GB", "48GB", "64GB"),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the instance. Depend on your tenant configuration. One of [enterprise-db, ]" +
					"enterprise-ds, professional-db, professional-ds, free-db]",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("free-db"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("enterprise-db", "enterprise-ds", "professional-db", "professional-ds", "free-db"),
				},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Cloud provider. One of [gcp, aws, azure]",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("gcp"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("gcp", "aws", "azure"),
				},
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "Id of the tenant",
				Required:            true,
			},
			"connection_url": schema.StringAttribute{
				MarkdownDescription: "Bolt connection URL to the instance database",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username of the instance database",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password of the instance database",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of Neo4j",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("5"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"paused": schema.BoolAttribute{
				MarkdownDescription: "Whether database is paused",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"storage": schema.StringAttribute{
				MarkdownDescription: "Storage allocated to the instance",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "Information about source for the instance",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"instance_id": schema.StringAttribute{
						MarkdownDescription: "Instance Id that contains the source database of the instance",
						Required:            true,
					},
					"snapshot_id": schema.StringAttribute{
						MarkdownDescription: "Snapshot Id of the instance containing the source database of the instance",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *InstanceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	postInstanceRequest := &client.PostInstanceRequest{
		Version:       data.Version.ValueString(),
		Region:        data.Region.ValueString(),
		Memory:        data.Memory.ValueString(),
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		TenantId:      data.TenantId.ValueString(),
		CloudProvider: data.CloudProvider.ValueString(),
	}
	if !data.Source.IsNull() {
		var sourceData InstanceResourceSourceModel
		response.Diagnostics.Append(data.Source.As(ctx, &sourceData, basetypes.ObjectAsOptions{})...)
		if response.Diagnostics.HasError() {
			return
		}
		postInstanceRequest.SourceInstanceId = sourceData.InstanceId.ValueStringPointer()
		if !sourceData.SnapshotId.IsNull() {
			_, err := r.auraApi.WaitUntilSnapshotIsInState(ctx, sourceData.InstanceId.ValueString(), sourceData.SnapshotId.ValueString(),
				func(resp client.GetSnapshotData) bool {
					return strings.ToLower(resp.Status) == "completed"
				})
			if err != nil {
				response.Diagnostics.AddError("Error while waiting snapshot to be completed", err.Error())
			}
			postInstanceRequest.SourceSnapshotId = sourceData.SnapshotId.ValueStringPointer()
		}
	}

	postInstanceResp, err := r.auraApi.PostInstance(*postInstanceRequest)
	if err != nil {
		response.Diagnostics.AddError("Error while creating an instance", err.Error())
		return
	}

	data.InstanceId = types.StringValue(postInstanceResp.Data.Id)
	data.ConnectionUrl = types.StringValue(postInstanceResp.Data.ConnectionUrl)
	data.Username = types.StringValue(postInstanceResp.Data.Username)
	data.Password = types.StringValue(postInstanceResp.Data.Password)

	tflog.Debug(ctx, "Created an instance with id "+postInstanceResp.Data.Id)

	instance, err := r.auraApi.WaitUntilInstanceIsInState(ctx, postInstanceResp.Data.Id, func(r client.GetInstanceResponse) bool {
		return strings.ToLower(r.Data.Status) == "running"
	})
	if err != nil {
		response.Diagnostics.AddError("Instance is not running in time", err.Error())
	}
	if instance.Data.Storage != nil {
		data.Storage = types.StringValue(*instance.Data.Storage)
	}

	tflog.Debug(ctx, fmt.Sprintf("Instance %s is running", postInstanceResp.Data.Id))

	// Pausing new instance
	if data.Paused.ValueBool() {
		diagError := r.pauseInstance(ctx, data.InstanceId.ValueString())
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

// todo implement override based on source instance
func (r *InstanceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan InstanceResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)

	if response.Diagnostics.HasError() {
		return
	}

	instance, err := r.auraApi.GetInstanceById(plan.InstanceId.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while getting instance details", err.Error())
		return
	}

	// Resume
	if !plan.Paused.ValueBool() && strings.ToLower(instance.Data.Status) == "paused" {
		diagError := r.resumeInstance(ctx, instance.Data.Id)
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	// Regular inplace update
	if plan.Name.ValueString() != instance.Data.Name || plan.Memory.ValueString() != instance.Data.Memory {
		tflog.Debug(ctx, fmt.Sprintf("Updating instance details: Name: %s -> %s. Memory: %s -> %s",
			instance.Data.Name, plan.Name.ValueString(), instance.Data.Memory, plan.Memory.ValueString()))
		_, err := r.auraApi.PatchInstanceById(instance.Data.Id, client.PatchInstanceRequest{
			Name:   plan.Name.ValueStringPointer(),
			Memory: plan.Memory.ValueStringPointer(),
		})

		if err != nil {
			response.Diagnostics.AddError("Error while updating the instance details", err.Error())
			return
		}

		_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, plan.InstanceId.ValueString(), func(resp client.GetInstanceResponse) bool {
			return resp.Data.Memory == plan.Memory.ValueString() &&
				resp.Data.Name == plan.Name.ValueString() &&
				(strings.ToLower(resp.Data.Status) == "running" || strings.ToLower(instance.Data.Status) == "paused")
		})

		if err != nil {
			response.Diagnostics.AddError("Error while waiting fro the instance details to be updated", err.Error())
			return
		}
	}

	// Pause
	if plan.Paused.ValueBool() && strings.ToLower(instance.Data.Status) != "paused" {
		diagError := r.pauseInstance(ctx, instance.Data.Id)
		if diagError.IsNotEmpty() {
			response.Diagnostics.AddError(diagError.Message, diagError.Details)
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.auraApi.DeleteInstanceById(data.InstanceId.ValueString())
	// todo should we wait until instance is deleted
	if err != nil {
		response.Diagnostics.AddError("Error while deleting an instance", err.Error())
	}
}

func (r *InstanceResource) resumeInstance(ctx context.Context, id string) util.DiagnosticsError {
	_, err := r.auraApi.ResumeInstanceById(id)
	if err != nil {
		return util.NewDiagnosticsError("Error while resume the instance", err.Error())
	}
	_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, id, func(resp client.GetInstanceResponse) bool {
		return strings.ToLower(resp.Data.Status) == "running"
	})
	if err != nil {
		return util.NewDiagnosticsError("Error while waiting instance to be resumed", err.Error())
	}
	return util.NoDiagnosticsError()
}

func (r *InstanceResource) pauseInstance(ctx context.Context, id string) util.DiagnosticsError {
	_, err := r.auraApi.PauseInstanceById(id)
	if err != nil {
		return util.NewDiagnosticsError("Error while pausing the instance", err.Error())
	}
	_, err = r.auraApi.WaitUntilInstanceIsInState(ctx, id, func(resp client.GetInstanceResponse) bool {
		return strings.ToLower(resp.Data.Status) == "paused"
	})
	if err != nil {
		return util.NewDiagnosticsError("Error while waiting for instance to be paused", err.Error())
	}
	return util.NoDiagnosticsError()
}
