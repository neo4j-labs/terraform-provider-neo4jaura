package client

import (
	"encoding/json"
	"errors"
	"fmt"
)

type AuraApi struct {
	auraClient *AuraClient
}

func NewAuraApi(client *AuraClient) *AuraApi {
	return &AuraApi{client}
}

func (api *AuraApi) GetTenants() (GetTenantsResponse, error) {
	payload, status, err := api.auraClient.Get("tenants")
	if err != nil {
		return GetTenantsResponse{}, err
	}

	if status != 200 {
		return GetTenantsResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(payload)))
	}

	var tenantsResponse GetTenantsResponse
	err = json.Unmarshal(payload, &tenantsResponse)
	if err != nil {
		return GetTenantsResponse{}, err
	}
	return tenantsResponse, err
}

func (api *AuraApi) PostInstance(request PostInstanceRequest) (PostInstanceResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return PostInstanceResponse{}, err
	}

	body, status, err := api.auraClient.Post("instances", payload)
	if err != nil {
		return PostInstanceResponse{}, err
	}

	if status != 202 {
		return PostInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(payload)))
	}

	var response PostInstanceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return PostInstanceResponse{}, err
	}
	return response, nil
}

func (api *AuraApi) GetInstanceById(id string) (GetInstanceResponse, error) {
	payload, status, err := api.auraClient.Get("instances/" + id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 200 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(payload)))
	}

	var response GetInstanceResponse
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	return response, nil
}

func (api *AuraApi) DeleteInstanceById(id string) (GetInstanceResponse, error) {
	payload, status, err := api.auraClient.Delete("instances/" + id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(payload)))
	}

	var response GetInstanceResponse
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	return response, nil
}

func (api *AuraApi) PatchInstanceById(id string, request PatchInstanceRequest) (GetInstanceResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return GetInstanceResponse{}, err
	}

	body, status, err := api.auraClient.Patch("instances/"+id, payload)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(body)))
	}

	var response GetInstanceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	return response, nil
}
