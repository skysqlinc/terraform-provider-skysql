package provider

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
)

func TestServiceResourceTags(t *testing.T) {
	const serviceID = "dbdgf42002418"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	tests := []struct {
		name         string
		testResource string
		before       func(r *require.Assertions)
		checks       []resource.TestCheckFunc
		expectError  *regexp.Regexp
	}{
		{
			name: "create service with tags",
			testResource: `
resource "skysql_service" "default" {
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
  deletion_protection = false
  tags = {
    "name" = "test-gcp"  # Must match service name (API overwrites this)
    "environment" = "development"
    "team" = "engineering"
  }
}
`,
			before: func(r *require.Assertions) {
				configureOnce.Reset()
				var service *provisioning.Service
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodGet, req.Method)
					r.Equal("/provisioning/v1/versions", req.URL.Path)
					r.Equal("page_size=1", req.URL.RawQuery)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode([]provisioning.Version{})
				})
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
							},
						},
						StorageVolume: struct {
							Size       int    `json:"size"`
							VolumeType string `json:"volume_type"`
							IOPS       int    `json:"iops"`
							Throughput int    `json:"throughput"`
						}{
							Size:       int(payload.Storage),
							VolumeType: "pd-ssd",
							IOPS:       int(payload.VolumeIOPS),
							Throughput: int(payload.VolumeThroughput),
						},
						OutboundIps:        nil,
						IsActive:           true,
						ServiceType:        payload.ServiceType,
						ReplicationEnabled: false,
						PrimaryHost:        "",
						Tags: map[string]string{
							"name":        "test-gcp", // API overwrites tags.name with service.Name
							"environment": "development",
							"team":        "engineering",
						},
					}
					r.NoError(json.NewEncoder(w).Encode(service))
					w.WriteHeader(http.StatusCreated)
				})
				for i := 0; i <= 2; i++ {
					expectRequest(func(w http.ResponseWriter, req *http.Request) {
						r.Equal(http.MethodGet, req.Method)
						r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						service.Status = "ready"
						r.NoError(json.NewEncoder(w).Encode(service))
						w.WriteHeader(http.StatusOK)
					})
				}
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodDelete, req.Method)
					r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
					w.WriteHeader(http.StatusAccepted)
					w.Header().Set("Content-Type", "application/json")
				})
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodGet, req.Method)
					r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(&skysql.ErrorResponse{
						Code: http.StatusNotFound,
					})
				})
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp"),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "development"),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.team", "engineering"),
			},
		},
		{
			name: "create service without tags",
			testResource: `
resource "skysql_service" "default" {
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
  deletion_protection = false
}
`,
			before: func(r *require.Assertions) {
				configureOnce.Reset()
				var service *provisioning.Service
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodGet, req.Method)
					r.Equal("/provisioning/v1/versions", req.URL.Path)
					r.Equal("page_size=1", req.URL.RawQuery)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode([]provisioning.Version{})
				})
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
							},
						},
						StorageVolume: struct {
							Size       int    `json:"size"`
							VolumeType string `json:"volume_type"`
							IOPS       int    `json:"iops"`
							Throughput int    `json:"throughput"`
						}{
							Size:       int(payload.Storage),
							VolumeType: "pd-ssd",
							IOPS:       int(payload.VolumeIOPS),
							Throughput: int(payload.VolumeThroughput),
						},
						OutboundIps:        nil,
						IsActive:           true,
						ServiceType:        payload.ServiceType,
						ReplicationEnabled: false,
						PrimaryHost:        "",
						Tags:               map[string]string{"name": payload.Name}, // API always sets name tag
					}
					r.NoError(json.NewEncoder(w).Encode(service))
					w.WriteHeader(http.StatusCreated)
				})
				for i := 0; i <= 2; i++ {
					expectRequest(func(w http.ResponseWriter, req *http.Request) {
						r.Equal(http.MethodGet, req.Method)
						r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						service.Status = "ready"
						r.NoError(json.NewEncoder(w).Encode(service))
						w.WriteHeader(http.StatusOK)
					})
				}
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodDelete, req.Method)
					r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
					w.WriteHeader(http.StatusAccepted)
					w.Header().Set("Content-Type", "application/json")
				})
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodGet, req.Method)
					r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(&skysql.ErrorResponse{
						Code: http.StatusNotFound,
					})
				})
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp"),
			},
		},
		{
			name: "create service without tags but API sets default tags",
			testResource: `
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "test-gcp-with-defaults"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = "10.6.11-6-1"
  wait_for_creation = true
  wait_for_deletion = true
  deletion_protection = false
}
`,
			before: func(r *require.Assertions) {
				configureOnce.Reset()
				var service *provisioning.Service
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodGet, req.Method)
					r.Equal("/provisioning/v1/versions", req.URL.Path)
					r.Equal("page_size=1", req.URL.RawQuery)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode([]provisioning.Version{})
				})
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
							},
						},
						StorageVolume: struct {
							Size       int    `json:"size"`
							VolumeType string `json:"volume_type"`
							IOPS       int    `json:"iops"`
							Throughput int    `json:"throughput"`
						}{
							Size:       int(payload.Storage),
							VolumeType: "pd-ssd",
							IOPS:       int(payload.VolumeIOPS),
							Throughput: int(payload.VolumeThroughput),
						},
						OutboundIps:        nil,
						IsActive:           true,
						ServiceType:        payload.ServiceType,
						ReplicationEnabled: false,
						PrimaryHost:        "",
						// API sets default tags based on service name
						Tags: map[string]string{
							"name": payload.Name, // API automatically sets name tag
						},
					}
					r.NoError(json.NewEncoder(w).Encode(service))
					w.WriteHeader(http.StatusCreated)
				})
				for i := 0; i <= 2; i++ {
					expectRequest(func(w http.ResponseWriter, req *http.Request) {
						r.Equal(http.MethodGet, req.Method)
						r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						service.Status = "ready"
						r.NoError(json.NewEncoder(w).Encode(service))
						w.WriteHeader(http.StatusOK)
					})
				}
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodDelete, req.Method)
					r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
					w.WriteHeader(http.StatusAccepted)
					w.Header().Set("Content-Type", "application/json")
				})
				expectRequest(func(w http.ResponseWriter, req *http.Request) {
					r.Equal(http.MethodGet, req.Method)
					r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(&skysql.ErrorResponse{
						Code: http.StatusNotFound,
					})
				})
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp-with-defaults"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := require.New(t)
			test.before(r)
			resource.Test(t, resource.TestCase{
				IsUnitTest: true,
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"skysql": providerserver.NewProtocol6WithError(New("")()),
				},
				Steps: []resource.TestStep{
					{
						Config:      test.testResource,
						Check:       resource.ComposeAggregateTestCheckFunc(test.checks...),
						ExpectError: test.expectError,
					},
				},
			})
		})
	}
}

func TestServiceResourceTagsUpdate(t *testing.T) {
	const serviceID = "dbdgf42002418"

	testURL, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
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

	// Create service with initial tags
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
				},
			},
			StorageVolume: struct {
				Size       int    `json:"size"`
				VolumeType string `json:"volume_type"`
				IOPS       int    `json:"iops"`
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: "pd-ssd",
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			Tags: map[string]string{
				"name":        "test-gcp", // API overwrites tags.name with service.Name
				"environment": "development",
			},
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})

	// Get service status (creation wait)
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
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

	// Update service tags
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPatch, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/tags", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		// Verify the request body contains the updated tags
		var updateReq provisioning.UpdateServiceTagsRequest
		err := json.NewDecoder(req.Body).Decode(&updateReq)
		r.NoError(err)
		r.Equal("test-gcp", updateReq.Tags["name"])
		r.Equal("production", updateReq.Tags["environment"])
		r.Equal("backend", updateReq.Tags["team"])

		// Update the service tags
		service.Tags = updateReq.Tags
		w.WriteHeader(http.StatusOK)
	})

	// Get service status after update
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})

	// Read state after update
	for i := 0; i < 2; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

	// Delete service
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	// Confirm deletion
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
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
resource "skysql_service" "default" {
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
  wait_for_update   = true
  deletion_protection = false
  tags = {
    "name" = "test-gcp"  # Must match service name (API overwrites this)
    "environment" = "development"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp"),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "development"),
				}...),
			},
			{
				Config: `
resource "skysql_service" "default" {
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
  wait_for_update   = true
  deletion_protection = false
  tags = {
    "name" = "test-gcp"  # Must match service name (API overwrites this)
    "environment" = "production"
    "team" = "backend"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp"),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "production"),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.team", "backend"),
				}...),
			},
		},
	})
}

func TestServiceResourceDefaultTagsPreservation(t *testing.T) {
	const serviceID = "dbdgf42002418"

	testURL, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
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

	// Create service without tags - API sets default tags
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
				},
			},
			StorageVolume: struct {
				Size       int    `json:"size"`
				VolumeType string `json:"volume_type"`
				IOPS       int    `json:"iops"`
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: "pd-ssd",
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			// API sets default tags when none specified
			Tags: map[string]string{
				"name": payload.Name, // API automatically sets name tag
			},
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})

	// Get service status (creation wait)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		service.Status = "ready"
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})

	// Refresh state after creation (first step)
	for i := 0; i < 3; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

	// Storage update request (when storage changes from 100 to 200)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPatch, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/storage", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// Update the service storage
		service.StorageVolume.Size = 200
		w.WriteHeader(http.StatusOK)
	})

	// Get service status after storage update (update wait)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})

	// No tags update should be called since we don't specify tags in the second config
	// and the current implementation should preserve the API-set defaults

	// Read state after "update" (second step)
	for i := 0; i < 2; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

	// Delete service
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	// Confirm deletion
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
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
				// Step 1: Create service without tags (API sets defaults)
				Config: `
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "test-gcp-defaults"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
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
					// Verify that the API-set default tag is preserved in state
					resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp-defaults"),
				}...),
			},
			{
				// Step 2: Update service configuration (still no tags specified)
				// This should preserve the default tags set by API
				Config: `
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "test-gcp-defaults"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 200  # Changed storage to trigger an update
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
					resource.TestCheckResourceAttr("skysql_service.default", "storage", "200"),
					// Verify that the default tags are still preserved after update
					resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp-defaults"),
				}...),
			},
		},
	})
}

// TestServiceResourceTagsNameChangeNoRecreation tests that the ModifyPlan logic prevents inconsistency
// With the new API behavior, attempting to change service name via tags.name will not work because:
// The test demonstrates that:
// 1. ✅ No service recreation occurs (no POST/DELETE calls)
// 2. ✅ No PATCH request occurs (ModifyPlan prevents inconsistency)
// 3. ✅ Terraform understands that tags.name will be overwritten by the API
// 4. ✅ Service name remains unchanged (top-level name wins)
func TestServiceResourceTagsApiOverwritesBehavior(t *testing.T) {
	// Test validates that the plan modifier prevents inconsistencies when
	// user mistakenly sets tags.name different from service name

	const serviceID = "dbdgf42002418"

	testURL, expectRequest, closeAPI := mockSkySQLAPI(t)
	defer closeAPI()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
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

	// Create service - API sets default name tag
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
					Mechanism:       "nlb",
					EndpointService: "",
					AllowList:       []provisioning.AllowListItem{},
				},
			},
			StorageVolume: struct {
				Size       int    `json:"size"`
				VolumeType string `json:"volume_type"`
				IOPS       int    `json:"iops"`
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: "pd-ssd",
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			// API automatically sets name tag from service name
			Tags: map[string]string{
				"name": payload.Name,
			},
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})

	// Get service status (creation wait)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		service.Status = "ready"
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})

	// Refresh state after creation (first step)
	for i := 0; i < 3; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

	// With the plan modifier, Terraform now automatically corrects tags.name
	// to match the service name during planning, so no PATCH request occurs
	// This validates that our plan modifier prevents the inconsistency

	// GET request for state refresh before delete
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
		w.WriteHeader(http.StatusOK)
	})

	// Delete service
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	// Confirm deletion
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
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
				// Step 1: Create service without explicit tags (API sets default name tag)
				Config: `
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "original-service"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
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
					resource.TestCheckResourceAttr("skysql_service.default", "name", "original-service"),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "original-service"),
				}...),
			},
			{
				// Step 2: Try to rename service via tags.name - API will overwrite tags.name with top-level name
				// This tests new API behavior: top-level name always wins and overwrites tags.name
				Config: `
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "es-single"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "original-service"  # Terraform config name stays same
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = "10.6.11-6-1"
  wait_for_creation = true
  wait_for_deletion = true
  wait_for_update   = true
  deletion_protection = false
  tags = {
    "name" = "renamed-service"  # Attempt to rename via tags (will be overwritten by API)
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),                 // Same ID = no recreation
					resource.TestCheckResourceAttr("skysql_service.default", "name", "original-service"),      // Name stays same (top-level wins)
					resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "original-service"), // tags.name overwritten by API
				}...),
			},
		},
	})
}
