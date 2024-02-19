package client

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiredIn   int64  `json:"expires_in"`
}

type TenantsResponse struct {
	Data []TenantsRepostData `json:"data"`
}

type TenantsRepostData struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
