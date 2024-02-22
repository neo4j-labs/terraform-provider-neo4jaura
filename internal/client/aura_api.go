package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/util"
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

	return util.Unmarshal[GetTenantsResponse](payload)
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
		return PostInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(body)))
	}

	return util.Unmarshal[PostInstanceResponse](body)
}

func (api *AuraApi) GetInstanceById(id string) (GetInstanceResponse, error) {
	payload, status, err := api.auraClient.Get("instances/" + id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 200 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(payload)))
	}
	return util.Unmarshal[GetInstanceResponse](payload)
}

func (api *AuraApi) DeleteInstanceById(id string) (GetInstanceResponse, error) {
	payload, status, err := api.auraClient.Delete("instances/" + id)
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(payload)))
	}
	return util.Unmarshal[GetInstanceResponse](payload)
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
	return util.Unmarshal[GetInstanceResponse](body)
}

func (api *AuraApi) PauseInstanceById(id string) (GetInstanceResponse, error) {
	body, status, err := api.auraClient.Post(fmt.Sprintf("instances/%s/pause", id), []byte("{}"))
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(body)))
	}
	return util.Unmarshal[GetInstanceResponse](body)
}

func (api *AuraApi) ResumeInstanceById(id string) (GetInstanceResponse, error) {
	body, status, err := api.auraClient.Post(fmt.Sprintf("instances/%s/resume", id), []byte("{}"))
	if err != nil {
		return GetInstanceResponse{}, err
	}
	if status != 202 {
		return GetInstanceResponse{}, errors.New("Aura error: " + fmt.Sprintf("Status: %+v. Response: %+v", status, string(body)))
	}
	return util.Unmarshal[GetInstanceResponse](body)
}

func (api *AuraApi) GetSnapshotsByInstanceId(instanceId string) (GetSnapshotsResponse, error) {
	body, status, err := api.auraClient.Get(fmt.Sprintf("instances/%s/snapshots", instanceId))
	if err != nil {
		return GetSnapshotsResponse{}, err
	}
	if status != 200 {
		return GetSnapshotsResponse{}, err
	}
	return util.Unmarshal[GetSnapshotsResponse](body)
}

func (api *AuraApi) GetSnapshotById(instanceId string, snapshotId string) (GetSnapshotResponse, error) {
	body, status, err := api.auraClient.Get(fmt.Sprintf("instances/%s/snapshots/%s", instanceId, snapshotId))
	if err != nil {
		return GetSnapshotResponse{}, err
	}
	if status != 200 {
		return GetSnapshotResponse{}, err
	}
	return util.Unmarshal[GetSnapshotResponse](body)
}
