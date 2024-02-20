package client

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiredIn   int64  `json:"expires_in"`
}

type GetTenantsResponse struct {
	Data []TenantsRepostData `json:"data"`
}

type TenantsRepostData struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type PostInstanceResponse struct {
	Data PostInstanceData `json:"data"`
}

type PostInstanceData struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	TenantId      string `json:"tenant_id"`
	CloudProvider string `json:"cloud_provider"`
	ConnectionUrl string `json:"connection_url"`
	Region        string `json:"region"`
	Type          string `json:"type"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type GetInstanceResponse struct {
	Data GetInstanceData `json:"data"`
}

// todo optional fields
type GetInstanceData struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	TenantId      string `json:"tenant_id"`
	CloudProvider string `json:"cloud_provider"`
	ConnectionUrl string `json:"connection_url"`
	Region        string `json:"region"`
	Type          string `json:"type"`
	Memory        string `json:"memory"`
}
