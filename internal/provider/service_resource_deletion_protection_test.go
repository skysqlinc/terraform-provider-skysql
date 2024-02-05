package provider

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestServiceResourceDeletionProtection(t *testing.T) {
	const serviceID = "dbdgf42002418"

	testURL, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_ACCESS_TOKEN", "[token]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testURL)

	r := require.New(t)

	configureOnce.Reset()
	var service *provisioning.Service
	// Check API connectivity
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		r.Equal("page_size=1", req.URL.RawQuery)
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
		service = &provisioning.Service{
			ID:           serviceID,
			Name:         payload.Name,
			Region:       payload.Region,
			Provider:     payload.Provider,
			Tier:         "foundation",
			Topology:     payload.Topology,
			Version:      payload.Version,
			Architecture: payload.Architecture,
			Size:         payload.Size,
			Nodes:        int(payload.Nodes),
			SSLEnabled:   payload.SSLEnabled,
			NosqlEnabled: payload.NoSQLEnabled,
			FQDN:         "",
			Status:       "pending_create",
			CreatedOn:    int(time.Now().Unix()),
			UpdatedOn:    int(time.Now().Unix()),
			CreatedBy:    uuid.New().String(),
			UpdatedBy:    uuid.New().String(),
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
					Mechanism:       payload.Mechanism,
					AllowedAccounts: payload.AllowedAccounts,
				},
			},
			StorageVolume: struct {
				Size       int    `json:"size"`
				VolumeType string `json:"volume_type"`
				IOPS       int    `json:"iops"`
			}{
				Size:       int(payload.Storage),
				VolumeType: payload.VolumeType,
				IOPS:       int(payload.VolumeIOPS),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			MaxscaleSize:       &(payload.Size),
			MaxscaleNodes:      0,
		}
		if service.Provider == "gcp" {
			service.StorageVolume.VolumeType = "pd-ssd"
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})
	for i := 0; i < 4; i++ {
		// Get service status
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}
	// Update service endpoint
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(
			fmt.Sprintf("%s %s/%s/endpoints", http.MethodPatch, "/provisioning/v1/services", serviceID),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		service.Endpoints[0].Mechanism = "privatelink"
		service.Endpoints[0].AllowedAccounts = []string{"mdb-cnewport"}
		service.Endpoints[0].Visibility = "private"
		json.NewEncoder(w).Encode(&provisioning.PatchServiceEndpointsResponse{
			{
				Mechanism:       service.Endpoints[0].Mechanism,
				AllowedAccounts: service.Endpoints[0].AllowedAccounts,
				Visibility:      service.Endpoints[0].Visibility,
			},
		})
		w.WriteHeader(http.StatusOK)
	})
	for i := 0; i < 4; i++ {
		// Get service status
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(
			fmt.Sprintf("%s %s/%s/power", http.MethodPost, "/provisioning/v1/services", serviceID),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 6; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(
				fmt.Sprintf("%s %s/%s", http.MethodGet, "/provisioning/v1/services", serviceID),
				fmt.Sprintf("%s %s", req.Method, req.URL.Path))
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			service.IsActive = false
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

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
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "test-gcp"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = "10.6.11-6-1"
  wait_for_creation = true
  wait_for_deletion = true
}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "deletion_protection", "true"),
				}...),
			},
			{
				Config: `
			resource "skysql_service" default {
			service_type   = "transactional"
			topology       = "es-single"
			cloud_provider = "gcp"
			region         = "us-central1"
			name           = "test-gcp"
			architecture   = "amd64"
			nodes          = 1
			size           = "sky-2x8"
			storage        = 100
			ssl_enabled    = true
			version        = "10.6.11-6-1"
			wait_for_creation = true
			wait_for_deletion = true
			wait_for_update = true
			endpoint_mechanism      = "privatelink"
			endpoint_allowed_accounts = ["mdb-cnewport"]
			deletion_protection = false
			}
				            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "deletion_protection", "false"),
				}...),
			},
			{
				Config: `
			resource "skysql_service" default {
			service_type   = "transactional"
			topology       = "es-single"
			cloud_provider = "gcp"
			region         = "us-central1"
			name           = "test-gcp"
			architecture   = "amd64"
			nodes          = 1
			size           = "sky-2x8"
			storage        = 100
			ssl_enabled    = true
			version        = "10.6.11-6-1"
			wait_for_creation = true
			wait_for_deletion = true
			wait_for_update = true
			endpoint_mechanism      = "privatelink"
			endpoint_allowed_accounts = ["mdb-cnewport"]
			is_active      = false
			}
				            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "deletion_protection", "true"),
				}...),
			},
			{
				Config: `
			resource "skysql_service" default {
			service_type   = "transactional"
			topology       = "es-single"
			cloud_provider = "gcp"
			region         = "us-central1"
			name           = "test-gcp"
			architecture   = "amd64"
			nodes          = 1
			size           = "sky-2x8"
			storage        = 100
			ssl_enabled    = true
			version        = "10.6.11-6-1"
			wait_for_creation = true
			wait_for_deletion = true
			wait_for_update = true
			endpoint_mechanism      = "privatelink"
			endpoint_allowed_accounts = ["mdb-cnewport"]
			is_active      = false
			deletion_protection = false
			}
				            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
				}...),
			},
		},
	})
}
