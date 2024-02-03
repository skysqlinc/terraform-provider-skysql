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
	"regexp"
	"testing"
	"time"
)

func TestServiceResourceServerlessAnalytics(t *testing.T) {
	const serviceID = "dbdgf42002418"

	testUrl, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_ACCESS_TOKEN", "[token]")
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
			Version:      "(none)",
			Architecture: "amd64",
			Size:         "",
			Nodes:        0,
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
							Name:    "sparksql",
							Port:    1000,
							Purpose: "sparksql",
						},
						{
							Name:    "https",
							Port:    443,
							Purpose: "https",
						},
						{
							Name:    "http",
							Port:    80,
							Purpose: "http",
						},
					},
					Visibility: "public",
					Mechanism:  "nlb",
				},
			},
			IsActive:    true,
			ServiceType: payload.ServiceType,
		}
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusCreated)
	})
	for i := 0; i < 8; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(
				fmt.Sprintf("%s %s/%s", http.MethodGet, "/provisioning/v1/services", serviceID),
				fmt.Sprintf("%s %s", req.Method, req.URL.Path))
			w.Header().Set("Content-Type", "application/json")
			service.Status = "ready"
			service.IsActive = true
			json.NewEncoder(w).Encode(&service)
			w.WriteHeader(http.StatusOK)
		})
	}
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(
			fmt.Sprintf("%s %s/%s", http.MethodDelete, "/provisioning/v1/services", serviceID),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		service.Status = "ready"
		json.NewEncoder(w).Encode(&service)
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
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
                  nodes = 1
			}
							            `,
				ExpectError: regexp.MustCompile(`The argument "nodes" is read only for the "sa" topology`),
				Destroy:     false,
			},
			{
				Config: `
			variable "architecture" {
			  type                 = string
			  default              = "amd64"
			}
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  architecture      = var.architecture
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
	            `,
				ExpectError: regexp.MustCompile(`The argument "architecture" is read only for the "sa" topology`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  size              = "sky-2x8"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
	            `,
				ExpectError: regexp.MustCompile(`The argument "size" is read only for the "sa" topology`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  ssl_enabled       = true
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
	            `,
				ExpectError: regexp.MustCompile(`The argument "ssl_enabled" is read only for the "sa" topology`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  version           = "8.0.23"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
	            `,
				ExpectError: regexp.MustCompile(`The argument "version" is read only for the "sa" topology`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
			   `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
				}...),
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
                  architecture      = "arm64"
				  name              = "serverless-analytics"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
			   `,
				ExpectError: regexp.MustCompile(`Cannot change service architecture`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
                  nodes             = 2
				  name              = "serverless-analytics"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
			   `,
				ExpectError: regexp.MustCompile(`Attempt to modify read-only attribute`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
                  size             =  "sky-2x8"
				  name              = "serverless-analytics"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
			   `,
				ExpectError: regexp.MustCompile(`Attempt to modify read-only attribute`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
                  ssl_enabled       =  true
				  name              = "serverless-analytics"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
			   `,
				ExpectError: regexp.MustCompile(`Cannot change service ssl_enabled`),
				Destroy:     false,
			},
			{
				Config: `
			resource "skysql_service" default {
				  service_type      = "analytical"
				  topology          = "sa"
				  cloud_provider    = "aws"
				  region            = "us-east-2"
				  name              = "serverless-analytics"
				  version           = "8.0.23"
				  wait_for_creation = true
				  wait_for_deletion = true
				  wait_for_update   = true
				  deletion_protection = false
			}
			  `,
				ExpectError: regexp.MustCompile(`Cannot change service version`),
				Destroy:     false,
			},
		},
	})
}
