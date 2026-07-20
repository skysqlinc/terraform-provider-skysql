package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
	"github.com/stretchr/testify/require"
)

// TestServiceResourceMaxscaleScale verifies that changing maxscale_nodes
// updates the service in place through POST /services/{id}/nodes with a
// maxscale-only payload, instead of destroying and recreating the service.
func TestServiceResourceMaxscaleScale(t *testing.T) {
	const serviceID = "dbdgf42002420"

	testUrl, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	r := require.New(t)

	configureOnce.Reset()
	var service *provisioning.Service
	// Check API connectivity
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})
	// Create service
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		payload := provisioning.CreateServiceRequest{}
		err := json.NewDecoder(req.Body).Decode(&payload)
		r.NoError(err)
		r.Equal(uint(1), payload.MaxscaleNodes)
		service = &provisioning.Service{
			ID:            serviceID,
			Name:          payload.Name,
			Region:        payload.Region,
			Provider:      payload.Provider,
			Tier:          "power",
			Topology:      payload.Topology,
			Version:       payload.Version,
			Architecture:  payload.Architecture,
			Size:          payload.Size,
			Nodes:         int(payload.Nodes),
			MaxscaleNodes: payload.MaxscaleNodes,
			MaxscaleSize:  payload.MaxscaleSize,
			SSLEnabled:    payload.SSLEnabled,
			Status:        "pending_create",
			CreatedOn:     int(time.Now().Unix()),
			UpdatedOn:     int(time.Now().Unix()),
			CreatedBy:     uuid.New().String(),
			UpdatedBy:     uuid.New().String(),
			Endpoints: []provisioning.Endpoint{
				{
					Name: "primary",
					Ports: []provisioning.Port{
						{
							Name:    "readwrite",
							Port:    3306,
							Purpose: "readwrite",
						},
					},
				},
			},
			StorageVolume: struct {
				Size       int    `json:"size"`
				VolumeType string `json:"volume_type"`
				IOPS       int    `json:"iops"`
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: payload.VolumeType,
				IOPS:       int(payload.VolumeIOPS),
			},
			IsActive:    true,
			ServiceType: payload.ServiceType,
		}
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusCreated)
	})
	// Get service status
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		service.Status = "ready"
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})
	// Refresh state
	for i := 0; i < 3; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(
				fmt.Sprintf("%s %s/%s", http.MethodGet, "/provisioning/v1/services", serviceID),
				fmt.Sprintf("%s %s", req.Method, req.URL.Path))
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}
	// Update maxscale nodes: the payload must carry only maxscale_nodes
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(
			fmt.Sprintf("%s %s/%s/nodes", http.MethodPost, "/provisioning/v1/services", serviceID),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		payload := &provisioning.UpdateServiceNodesNumberRequest{}
		err := json.NewDecoder(req.Body).Decode(payload)
		r.NoError(err)
		r.Zero(payload.Nodes)
		r.NotNil(payload.MaxscaleNodes)
		r.Equal(int64(2), *payload.MaxscaleNodes)
		service.MaxscaleNodes = uint(*payload.MaxscaleNodes)
		w.WriteHeader(http.StatusOK)
	})
	for i := 0; i < 3; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(
				fmt.Sprintf("%s %s/%s", http.MethodGet, "/provisioning/v1/services", serviceID),
				fmt.Sprintf("%s %s", req.Method, req.URL.Path))
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}
	// Delete service
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(
			fmt.Sprintf("%s %s/%s", http.MethodDelete, "/provisioning/v1/services", serviceID),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		service.Status = "ready"
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(
			fmt.Sprintf("%s %s/%s", http.MethodGet, "/provisioning/v1/services", serviceID),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&skysql.ErrorResponse{
			Code: http.StatusNotFound,
		})
	})

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
			resource "skysql_service" default {
				  service_type   = "transactional"
				  topology       = "galera"
				  cloud_provider = "aws"
				  region         = "us-east-2"
				  name           = "my-service"
				  architecture   = "amd64"
				  nodes          = 3
				  maxscale_nodes = 1
				  maxscale_size  = "sky-2x4"
				  size           = "sky-4x16"
				  storage        = 100
				  volume_type    = "io1"
				  volume_iops    = 3000
				  ssl_enabled    = true
				  version        = "10.6.11-6-1"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "maxscale_nodes", "1"),
				}...),
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type   = "transactional"
				  topology       = "galera"
				  cloud_provider = "aws"
				  region         = "us-east-2"
				  name           = "my-service"
				  architecture   = "amd64"
				  nodes          = 3
				  maxscale_nodes = 2
				  maxscale_size  = "sky-2x4"
				  size           = "sky-4x16"
				  storage        = 100
				  volume_type    = "io1"
				  volume_iops    = 3000
				  ssl_enabled    = true
				  version        = "10.6.11-6-1"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "maxscale_nodes", "2"),
				}...),
			},
		},
	})
}
