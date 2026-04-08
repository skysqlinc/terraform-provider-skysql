package provider

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestServiceResourceServerlessStandalone_IsActiveReadOnly(t *testing.T) {
	const serviceID = "dbdgf42002419"

	testUrl, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	r := require.New(t)

	configureOnce.Reset()
	// Check API connectivity
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		r.Equal("page_size=1", req.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})
	var service *provisioning.Service
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
			Version:      "11.4.2",
			Architecture: "amd64",
			Size:         "sky-2x8",
			Nodes:        1,
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
					Visibility: "public",
					Mechanism:  "nlb",
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
			SSLEnabled:  payload.SSLEnabled,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(service)
	})
	// Multiple GET requests for service status checks during and after creation
	for i := 0; i < 4; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal(fmt.Sprintf("/provisioning/v1/services/%s", serviceID), req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			service.IsActive = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(service)
		})
	}

	// Delete service
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodDelete, req.Method)
		r.Equal(fmt.Sprintf("/provisioning/v1/services/%s", serviceID), req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(service)
	})

	// Verify deletion (404)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal(fmt.Sprintf("/provisioning/v1/services/%s", serviceID), req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": http.StatusNotFound,
		})
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			// Test 1: Attempt to set is_active during creation - should fail
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "transactional"
				  topology          = "serverless-standalone"
				  cloud_provider    = "aws"
				  region            = "us-east-1"
				  name              = "sls-standalone-test"
				  is_active         = true
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
				  ssl_enabled       = true
				  size              = "sky-2x8"
				  storage           = 100
				  volume_type       = "io1"
				  volume_iops       = 3000
			}
							            `,
				ExpectError: regexp.MustCompile(`Start/stop operations are not supported for serverless services`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "transactional"
				  topology          = "serverless-standalone"
				  cloud_provider    = "aws"
				  region            = "us-east-1"
				  name              = "sls-standalone-test"
				  is_active         = false
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
				  ssl_enabled       = true
				  size              = "sky-2x8"
				  storage           = 100
				  volume_type       = "io1"
				  volume_iops       = 3000
			}
							            `,
				ExpectError: regexp.MustCompile(`Start/stop operations are not supported for serverless services`),
				Destroy:     false,
			},
			// Test 2: Create without is_active - should succeed
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "transactional"
				  topology          = "serverless-standalone"
				  cloud_provider    = "aws"
				  region            = "us-east-1"
				  name              = "sls-standalone-test"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
				  ssl_enabled       = true
				  size              = "sky-2x8"
				  storage           = 100
				  volume_type       = "io1"
				  volume_iops       = 3000
			}
			   `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "is_active", "true"),
				}...),
			},
			// Test 3: Attempt to change is_active during update - should fail
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "transactional"
				  topology          = "serverless-standalone"
				  cloud_provider    = "aws"
				  region            = "us-east-1"
				  name              = "sls-standalone-test"
				  is_active         = false
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
				  ssl_enabled       = true
				  size              = "sky-2x8"
				  storage           = 100
				  volume_type       = "io1"
				  volume_iops       = 3000
			}
			   `,
				ExpectError: regexp.MustCompile(`Start/stop operations are not supported for serverless services`),
				Destroy:     false,
			},
		},
	})
}
