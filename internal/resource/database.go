/*
 *  Copyright (c) "Neo4j"
 *  Neo4j Sweden AB [https://neo4j.com]
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package resource

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/neo4j-labs/terraform-provider-neo4jaura/internal/client"
)

var (
	_ resource.Resource                = &DatabaseResource{}
	_ resource.ResourceWithConfigure   = &DatabaseResource{}
	_ resource.ResourceWithImportState = &DatabaseResource{}
)

func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

type DatabaseResource struct {
	auraApi *client.AuraApi
}

type DatabaseResourceModel struct {
	OrganizationId          types.String `tfsdk:"organization_id"`
	ProjectId               types.String `tfsdk:"project_id"`
	InstanceId              types.String `tfsdk:"instance_id"`
	DatabaseId              types.String `tfsdk:"database_id"`
	Name                    types.String `tfsdk:"name"`
	ReplicateFromDatabaseId types.String `tfsdk:"replicate_from_database_id"`
}

// Aura database names must match this pattern per the v2beta1 API spec
// (3-63 chars, lowercase letters, digits and dashes, starting with a letter
// and ending with a letter or digit).
var databaseNameRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)

// reservedDatabaseNames are the system/default databases every Aura instance
// provides on its own; they cannot be managed via Terraform.
var reservedDatabaseNames = []string{"neo4j", "system"}

// reservedDatabaseNameValidator rejects database names that collide with the
// system/default databases the platform creates automatically.
type reservedDatabaseNameValidator struct{}

func (v *reservedDatabaseNameValidator) Description(_ context.Context) string {
	return fmt.Sprintf("database name must not be one of the reserved system databases: %s", strings.Join(reservedDatabaseNames, ", "))
}

func (v *reservedDatabaseNameValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("Database name must not be one of the reserved system databases (`%s`). "+
		"The default databases are created and managed by the platform and cannot be managed via this resource.", strings.Join(reservedDatabaseNames, "`, `"))
}

func (v *reservedDatabaseNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	name := req.ConfigValue.ValueString()
	for _, reserved := range reservedDatabaseNames {
		if name == reserved {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Reserved database name",
				fmt.Sprintf("The name %q collides with a default database created and managed by the platform. "+
					"Default databases cannot be managed via this resource; choose a different name.", name),
			)
			return
		}
	}
}

func (r *DatabaseResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_database"
}

func (r *DatabaseResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	auraApi, ok := request.ProviderData.(*client.AuraApi)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AuraApi, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}
	r.auraApi = auraApi
}

func (r *DatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages a database within a Neo4j Aura instance (multi-database). " +
			"Requires an instance that supports multi-database.",
		Description: "Manages a database within a Neo4j Aura instance (multi-database). " +
			"Requires an instance that supports multi-database.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the organization that contains the project and instance.",
				Description:         "The ID of the organization that contains the project and instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the project that contains the instance.",
				Description:         "The ID of the project that contains the instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the instance to create the database in.",
				Description:         "The ID of the instance to create the database in.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the database, assigned by the Aura API.",
				Description:         "The ID of the database, assigned by the Aura API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The name of the database. 3-63 characters, lowercase letters, digits and dashes; " +
					"must start with a letter and end with a letter or digit. Changing the name forces recreation.",
				Description: "The name of the database. 3-63 characters, lowercase letters, digits and dashes; " +
					"must start with a letter and end with a letter or digit. Changing the name forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(databaseNameRegexp, "must be 3-63 characters of lowercase letters, digits "+
						"and dashes, starting with a letter and ending with a letter or digit"),
					&reservedDatabaseNameValidator{},
				},
			},
			"replicate_from_database_id": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The ID of an existing database on the same instance to replicate from. " +
					"Must be provided on its own. Changing this forces recreation.",
				Description: "The ID of an existing database on the same instance to replicate from. " +
					"Must be provided on its own. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DatabaseResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data DatabaseResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	instanceId := data.InstanceId.ValueString()
	name := data.Name.ValueString()

	createReq := client.PostDatabaseRequest{
		Name: name,
	}
	if !data.ReplicateFromDatabaseId.IsNull() && !data.ReplicateFromDatabaseId.IsUnknown() {
		createReq.ReplicateFromDatabaseId = data.ReplicateFromDatabaseId.ValueStringPointer()
	}

	createResp, err := r.auraApi.PostDatabase(ctx, orgId, projectId, instanceId, createReq)
	if err != nil {
		response.Diagnostics.AddError(
			"Error creating database",
			fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s: %s", orgId, projectId, instanceId, err.Error()),
		)
		return
	}

	data.DatabaseId = types.StringValue(createResp.Data.Id)

	// Read back to confirm the database is visible (delete-during-create and
	// other races surface here as a not-found error instead of silent drift).
	if _, err := r.auraApi.GetDatabaseById(ctx, orgId, projectId, instanceId, createResp.Data.Id); err != nil {
		response.Diagnostics.AddError(
			"Error reading database after creation",
			fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s database_id=%s: %s",
				orgId, projectId, instanceId, createResp.Data.Id, err.Error()),
		)
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *DatabaseResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data DatabaseResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	instanceId := data.InstanceId.ValueString()
	databaseId := data.DatabaseId.ValueString()

	// After an import by name, the database ID is not yet known — resolve it
	// from the instance's database list before reading the database itself.
	if databaseId == "" {
		listResp, err := r.auraApi.GetDatabasesByInstanceId(ctx, orgId, projectId, instanceId)
		if errors.Is(err, client.ErrNotFound) {
			response.State.RemoveResource(ctx)
			return
		}
		if err != nil {
			response.Diagnostics.AddError(
				"Error reading databases",
				fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s: %s", orgId, projectId, instanceId, err.Error()),
			)
			return
		}
		name := data.Name.ValueString()
		for _, db := range listResp.Data {
			if db.Name == name {
				databaseId = db.Id
				data.DatabaseId = types.StringValue(db.Id)
				break
			}
		}
		if databaseId == "" {
			// No database with the imported name exists anymore — remove from state.
			response.State.RemoveResource(ctx)
			return
		}
	}

	db, err := r.auraApi.GetDatabaseById(ctx, orgId, projectId, instanceId, databaseId)
	if errors.Is(err, client.ErrNotFound) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError(
			"Error reading database",
			fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s database_id=%s: %s",
				orgId, projectId, instanceId, databaseId, err.Error()),
		)
		return
	}

	data.Name = types.StringValue(db.Data.Name)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Update is intentionally minimal: every mutable attribute in the schema is
// marked RequiresReplace, so the framework replaces the resource instead of
// calling Update. The method exists to refresh computed fields if the
// framework ever does invoke it.
func (r *DatabaseResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data DatabaseResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	instanceId := data.InstanceId.ValueString()
	databaseId := data.DatabaseId.ValueString()

	db, err := r.auraApi.GetDatabaseById(ctx, orgId, projectId, instanceId, databaseId)
	if errors.Is(err, client.ErrNotFound) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError(
			"Error reading database",
			fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s database_id=%s: %s",
				orgId, projectId, instanceId, databaseId, err.Error()),
		)
		return
	}

	data.Name = types.StringValue(db.Data.Name)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *DatabaseResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data DatabaseResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	orgId := data.OrganizationId.ValueString()
	projectId := data.ProjectId.ValueString()
	instanceId := data.InstanceId.ValueString()
	databaseId := data.DatabaseId.ValueString()

	if err := r.auraApi.DeleteDatabaseById(ctx, orgId, projectId, instanceId, databaseId); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return
		}
		response.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s database_id=%s: %s",
				orgId, projectId, instanceId, databaseId, err.Error()),
		)
		return
	}

	// Deletion is asynchronous (202 Accepted) — poll until the database is gone.
	if err := r.auraApi.WaitUntilDatabaseIsDeleted(ctx, orgId, projectId, instanceId, databaseId); err != nil {
		response.Diagnostics.AddError(
			"Error waiting for database to be deleted",
			fmt.Sprintf("organization_id=%s project_id=%s instance_id=%s database_id=%s: %s",
				orgId, projectId, instanceId, databaseId, err.Error()),
		)
		return
	}
}

func (r *DatabaseResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	// The import ID uses the database *name* (not ID), since the name is what
	// the user knows: {organization_id}/{project_id}/{instance_id}/{database_name}.
	parts := strings.Split(request.ID, "/")
	if len(parts) != 4 {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id}/{project_id}/{instance_id}/{database_name}. Got: %q", request.ID),
		)
		return
	}
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		response.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: {organization_id}/{project_id}/{instance_id}/{database_name}. Got: %q", request.ID),
		)
		return
	}
	orgId, projectId, instanceId, databaseName := parts[0], parts[1], parts[2], parts[3]

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("organization_id"), orgId)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("instance_id"), instanceId)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("name"), databaseName)...)
	if response.Diagnostics.HasError() {
		return
	}
}
