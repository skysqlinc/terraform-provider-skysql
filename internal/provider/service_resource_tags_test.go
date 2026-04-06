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
			name: "create service with user-managed tags",
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
						// API returns all tags including the injected "name"
						Tags: map[string]string{
							"name":        "test-gcp",
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
				// Only user-managed tags should appear in state; API-injected "name" is filtered out
				resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "development"),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.team", "engineering"),
				resource.TestCheckNoResourceAttr("skysql_service.default", "tags.name"),
			},
		},
		{
			name: "create service without tags - API-injected tags are not tracked",
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
				// Tags not specified in config => tags should be null in state
				resource.TestCheckNoResourceAttr("skysql_service.default", "tags.%"),
			},
		},
		{
			name: "create service with explicit tags.name matching service name",
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
    "name" = "test-gcp"
    "environment" = "development"
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
							"name":        payload.Name,
							"environment": "development",
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
				// When user explicitly includes tags.name matching service name, it appears in state
				resource.TestCheckResourceAttr("skysql_service.default", "tags.name", "test-gcp"),
				resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "development"),
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
				"name":        "test-gcp", // API injects name
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

	// Update service tags PATCH
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPatch, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/tags", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		var updateReq provisioning.UpdateServiceTagsRequest
		err := json.NewDecoder(req.Body).Decode(&updateReq)
		r.NoError(err)
		r.Equal("production", updateReq.Tags["environment"])
		r.Equal("backend", updateReq.Tags["team"])

		// Update the service tags (API still injects name)
		service.Tags = map[string]string{
			"name":        "test-gcp",
			"environment": "production",
			"team":        "backend",
		}
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
    "environment" = "development"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "development"),
					resource.TestCheckNoResourceAttr("skysql_service.default", "tags.name"),
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
    "environment" = "production"
    "team" = "backend"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.environment", "production"),
					resource.TestCheckResourceAttr("skysql_service.default", "tags.team", "backend"),
					resource.TestCheckNoResourceAttr("skysql_service.default", "tags.name"),
				}...),
			},
		},
	})
}

// TestServiceResourceNoTagsUpdateOnOtherChanges tests that when tags are not specified,
// updating other fields (like storage) does not trigger a tag update.
func TestServiceResourceNoTagsUpdateOnOtherChanges(t *testing.T) {
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

	// Create service without tags
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
			// API sets default tags even when user didn't specify any
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

	// Refresh state after creation
	for i := 0; i < 3; i++ {
		expectRequest(func(w http.ResponseWriter, req *http.Request) {
			r.Equal(http.MethodGet, req.Method)
			r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(service)
			w.WriteHeader(http.StatusOK)
		})
	}

	// Storage update request (no tag update should happen)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r.Equal(http.MethodPatch, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/storage", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		service.StorageVolume.Size = 200
		w.WriteHeader(http.StatusOK)
	})

	// Get service status after storage update
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
					// Tags not specified => null in state
					resource.TestCheckNoResourceAttr("skysql_service.default", "tags.%"),
				}...),
			},
			{
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
  storage        = 200
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
					// Tags still null after storage update
					resource.TestCheckNoResourceAttr("skysql_service.default", "tags.%"),
				}...),
			},
		},
	})
}
