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

func TestServiceResourceGCPVolumeType(t *testing.T) {
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
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: payload.VolumeType,
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			MaxscaleSize:       &(payload.Size),
			MaxscaleNodes:      0,
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})
	for i := 0; i < 3; i++ {
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
					volume_type    = "pd-ssd"
					deletion_protection = "false"
				}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "volume_type", "pd-ssd"),
				}...),
			},
		},
	})
}

func TestServiceResourceAWSGP3VolumeType(t *testing.T) {
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
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: payload.VolumeType,
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			MaxscaleSize:       &(payload.Size),
			MaxscaleNodes:      0,
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})
	for i := 0; i < 3; i++ {
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
					cloud_provider = "aws"
					region         = "us-central1"
					name           = "test-gcp"
					architecture   = "amd64"
					nodes          = 1
					size           = "sky-2x8"
					storage        = 100
					ssl_enabled    = true
					version        = "10.6.11-6-1"
					volume_type    = "gp3"
					volume_iops    = 3000
					volume_throughput = 125
					deletion_protection = "false"
				}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "volume_type", "gp3"),
				}...),
			},
		},
	})
}

func TestServiceResourceAWSIO1VolumeType(t *testing.T) {
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
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: payload.VolumeType,
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			MaxscaleSize:       &(payload.Size),
			MaxscaleNodes:      0,
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})
	for i := 0; i < 3; i++ {
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
					cloud_provider = "aws"
					region         = "us-central1"
					name           = "test-gcp"
					architecture   = "amd64"
					nodes          = 1
					size           = "sky-2x8"
					storage        = 100
					ssl_enabled    = true
					version        = "10.6.11-6-1"
					volume_type    = "io1"
				    volume_iops    = 3000
					deletion_protection = "false"
				}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "volume_type", "io1"),
				}...),
			},
		},
	})
}

func TestServiceResourceAzureVolumeType(t *testing.T) {
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
				Throughput int    `json:"throughput"`
			}{
				Size:       int(payload.Storage),
				VolumeType: payload.VolumeType,
				IOPS:       int(payload.VolumeIOPS),
				Throughput: int(payload.VolumeThroughput),
			},
			OutboundIps:        nil,
			IsActive:           true,
			ServiceType:        payload.ServiceType,
			ReplicationEnabled: false,
			PrimaryHost:        "",
			MaxscaleSize:       &(payload.Size),
			MaxscaleNodes:      0,
		}
		r.NoError(json.NewEncoder(w).Encode(service))
		w.WriteHeader(http.StatusCreated)
	})
	for i := 0; i < 3; i++ {
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
					cloud_provider = "azure"
					region         = "us-central1"
					name           = "test-gcp"
					architecture   = "amd64"
					nodes          = 1
					size           = "sky-2x8"
					storage        = 100
					ssl_enabled    = true
					version        = "10.6.11-6-1"
					volume_type    = "StandardSSD_LRS"
					deletion_protection = "false"
				}
	            `,
				Check: resource.ComposeAggregateTestCheckFunc([]resource.TestCheckFunc{
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "volume_type", "StandardSSD_LRS"),
				}...),
			},
		},
	})
}
