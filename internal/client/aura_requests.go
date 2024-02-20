package client

// todo source instance id && source snapshot id
type PostInstanceRequest struct {
	Version       string `json:"version"`
	Region        string `json:"region"`
	Memory        string `json:"memory"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	TenantId      string `json:"tenant_id"`
	CloudProvider string `json:"cloud_provider"`
}

type PatchInstanceRequest struct {
	Name   *string `json:"name"`
	Memory *string `json:"memory"`
}
