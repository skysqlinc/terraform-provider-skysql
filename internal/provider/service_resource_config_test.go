package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
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

func TestServiceResourceWithConfigID(t *testing.T) {
	configureOnce.Reset()

	const serviceID = "dbdgf42002420"
	const configID = "cfg-test-uuid-001"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	service := &provisioning.Service{
		ID:           serviceID,
		Name:         "test-with-config",
		Region:       "us-central1",
		Provider:     "gcp",
		Tier:         "power",
		Topology:     "es-single",
		Version:      "10.6.11-6-1",
		Architecture: "amd64",
		Size:         "sky-2x8",
		Nodes:        1,
		SSLEnabled:   true,
		FQDN:         "",
		Status:       "ready",
		CreatedOn:    int(time.Now().Unix()),
		UpdatedOn:    int(time.Now().Unix()),
		CreatedBy:    uuid.New().String(),
		UpdatedBy:    uuid.New().String(),
		Endpoints: []provisioning.Endpoint{
			{
				Name: "primary",
				Ports: []provisioning.Port{
					{Name: "readwrite", Port: 3306, Purpose: "readwrite"},
				},
			},
		},
		StorageVolume: struct {
			Size       int    `json:"size"`
			VolumeType string `json:"volume_type"`
			IOPS       int    `json:"iops"`
			Throughput int    `json:"throughput"`
		}{Size: 100, VolumeType: "pd-ssd"},
		IsActive:    true,
		ServiceType: "transactional",
	}

	// 1. Provider configure: GET /versions
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})

	// 2. Create: POST /services
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		pendingService := *service
		pendingService.Status = "pending_create"
		json.NewEncoder(w).Encode(&pendingService)
	})

	// 3. Wait for creation: GET /services/{id} → ready
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})

	// 4. readServiceState after wait: GET /services/{id}
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})

	// 5. Apply config: POST /services/{id}/config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/config", req.URL.Path)

		var payload provisioning.ServiceConfigState
		err := json.NewDecoder(req.Body).Decode(&payload)
		r.NoError(err)
		r.Equal(configID, payload.ConfigID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(&payload)
	})

	// 6. Wait for config apply: GET /services/{id} → ready
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		serviceWithConfig := *service
		serviceWithConfig.ConfigID = configID
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})

	// 7. Terraform Read after Create: GET /services/{id}
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		serviceWithConfig := *service
		serviceWithConfig.ConfigID = configID
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})

	// 8. Destroy: DELETE /services/{id}
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.WriteHeader(http.StatusAccepted)
	})

	// 9. Wait for deletion: GET /services/{id} → 404
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&skysql.ErrorResponse{Code: http.StatusNotFound})
	})

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "skysql_service" "default" {
					service_type        = "transactional"
					topology            = "es-single"
					cloud_provider      = "gcp"
					region              = "us-central1"
					name                = "test-with-config"
					architecture        = "amd64"
					nodes               = 1
					size                = "sky-2x8"
					storage             = 100
					ssl_enabled         = true
					version             = "10.6.11-6-1"
					wait_for_creation   = true
					wait_for_deletion   = true
					deletion_protection = false
					config_id           = "%s"
				}`, configID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "config_id", configID),
				),
			},
		},
	})
}

func TestServiceResourceConfigID_WaitForCreationRequired(t *testing.T) {
	configureOnce.Reset()

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	// Provider configure: GET /versions
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
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
					service_type        = "transactional"
					topology            = "es-single"
					cloud_provider      = "gcp"
					region              = "us-central1"
					name                = "test-no-wait"
					architecture        = "amd64"
					nodes               = 1
					size                = "sky-2x8"
					storage             = 100
					ssl_enabled         = true
					version             = "10.6.11-6-1"
					wait_for_creation   = false
					wait_for_deletion   = true
					deletion_protection = false
					config_id           = "some-config-id"
				}`,
				ExpectError: regexp.MustCompile(`config_id requires wait_for_creation = true`),
			},
		},
	})
}

func TestServiceResourceConfigID_SameConfigNoOp(t *testing.T) {
	configureOnce.Reset()

	const serviceID = "dbdgf42002421"
	const configID = "cfg-already-applied"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	service := &provisioning.Service{
		ID:           serviceID,
		Name:         "test-same-config",
		Region:       "us-central1",
		Provider:     "gcp",
		Tier:         "power",
		Topology:     "es-single",
		Version:      "10.6.11-6-1",
		Architecture: "amd64",
		Size:         "sky-2x8",
		Nodes:        1,
		SSLEnabled:   true,
		FQDN:         "",
		Status:       "ready",
		CreatedOn:    int(time.Now().Unix()),
		UpdatedOn:    int(time.Now().Unix()),
		CreatedBy:    uuid.New().String(),
		UpdatedBy:    uuid.New().String(),
		Endpoints: []provisioning.Endpoint{
			{
				Name: "primary",
				Ports: []provisioning.Port{
					{Name: "readwrite", Port: 3306, Purpose: "readwrite"},
				},
			},
		},
		StorageVolume: struct {
			Size       int    `json:"size"`
			VolumeType string `json:"volume_type"`
			IOPS       int    `json:"iops"`
			Throughput int    `json:"throughput"`
		}{Size: 100, VolumeType: "pd-ssd"},
		IsActive:    true,
		ServiceType: "transactional",
	}

	// Step 1: Create service without config_id.

	// Provider configure
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})
	// Create: POST /services
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		pendingService := *service
		pendingService.Status = "pending_create"
		json.NewEncoder(w).Encode(&pendingService)
	})
	// Wait for creation: GET /services/{id} → ready
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// readServiceState after wait
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// Terraform Read after Create
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})

	// Step 2: Add config_id — but the service already has this config applied
	// (simulates import scenario where TF state is empty but service has config).

	// Terraform Read (pre-plan refresh)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		// Service already has the config applied (set externally or via import)
		serviceWithConfig := *service
		serviceWithConfig.ConfigID = configID
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})
	// Update: GET /services/{id} to check actual config (updateServiceConfig)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		serviceWithConfig := *service
		serviceWithConfig.ConfigID = configID
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})
	// readServiceState at end of Update
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		serviceWithConfig := *service
		serviceWithConfig.ConfigID = configID
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})
	// Terraform Read after Update
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		serviceWithConfig := *service
		serviceWithConfig.ConfigID = configID
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})

	// Destroy
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.WriteHeader(http.StatusAccepted)
	})
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&skysql.ErrorResponse{Code: http.StatusNotFound})
	})

	serviceHCLNoConfig := `
	resource "skysql_service" "default" {
		service_type        = "transactional"
		topology            = "es-single"
		cloud_provider      = "gcp"
		region              = "us-central1"
		name                = "test-same-config"
		architecture        = "amd64"
		nodes               = 1
		size                = "sky-2x8"
		storage             = 100
		ssl_enabled         = true
		version             = "10.6.11-6-1"
		wait_for_creation   = true
		wait_for_deletion   = true
		deletion_protection = false
	}`

	serviceHCLWithConfig := fmt.Sprintf(`
	resource "skysql_service" "default" {
		service_type        = "transactional"
		topology            = "es-single"
		cloud_provider      = "gcp"
		region              = "us-central1"
		name                = "test-same-config"
		architecture        = "amd64"
		nodes               = 1
		size                = "sky-2x8"
		storage             = 100
		ssl_enabled         = true
		version             = "10.6.11-6-1"
		wait_for_creation   = true
		wait_for_deletion   = true
		deletion_protection = false
		config_id           = "%s"
	}`, configID)

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: serviceHCLNoConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckNoResourceAttr("skysql_service.default", "config_id"),
				),
			},
			{
				// Add config_id that the service already has — should be a no-op
				// (no POST /services/{id}/config call — only GETs).
				Config: serviceHCLWithConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_service.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_service.default", "config_id", configID),
				),
			},
		},
	})
}

// TestServiceResourceConfigID_SwapConfig verifies that changing config_id from one
// config to another applies the new config via POST /services/{id}/config.
func TestServiceResourceConfigID_SwapConfig(t *testing.T) {
	configureOnce.Reset()

	const serviceID = "dbdgf42002422"
	const configA = "cfg-config-a"
	const configB = "cfg-config-b"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	service := &provisioning.Service{
		ID:           serviceID,
		Name:         "test-swap-config",
		Region:       "us-central1",
		Provider:     "gcp",
		Tier:         "power",
		Topology:     "es-single",
		Version:      "10.6.11-6-1",
		Architecture: "amd64",
		Size:         "sky-2x8",
		Nodes:        1,
		SSLEnabled:   true,
		Status:       "ready",
		CreatedOn:    int(time.Now().Unix()),
		UpdatedOn:    int(time.Now().Unix()),
		CreatedBy:    uuid.New().String(),
		UpdatedBy:    uuid.New().String(),
		Endpoints: []provisioning.Endpoint{
			{
				Name: "primary",
				Ports: []provisioning.Port{
					{Name: "readwrite", Port: 3306, Purpose: "readwrite"},
				},
			},
		},
		StorageVolume: struct {
			Size       int    `json:"size"`
			VolumeType string `json:"volume_type"`
			IOPS       int    `json:"iops"`
			Throughput int    `json:"throughput"`
		}{Size: 100, VolumeType: "pd-ssd"},
		IsActive:    true,
		ServiceType: "transactional",
	}

	serviceWithConfigA := *service
	serviceWithConfigA.ConfigID = configA

	serviceWithConfigB := *service
	serviceWithConfigB.ConfigID = configB

	// --- Step 1: Create service with config_id = configA ---

	// Provider configure
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})
	// Create: POST /services
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		w.Header().Set("Content-Type", "application/json")
		pendingService := *service
		pendingService.Status = "pending_create"
		json.NewEncoder(w).Encode(&pendingService)
	})
	// Wait for creation
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// readServiceState after wait
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// Apply config A: POST /services/{id}/config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/config", req.URL.Path)
		var payload provisioning.ServiceConfigState
		json.NewDecoder(req.Body).Decode(&payload)
		r.Equal(configA, payload.ConfigID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(&payload)
	})
	// Wait for config apply
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigA)
	})
	// Read after create
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigA)
	})

	// --- Step 2: Update config_id from configA to configB ---

	// Terraform Read (pre-plan refresh)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigA)
	})
	// Update: GET /services/{id} to check actual config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigA)
	})
	// Apply config B: POST /services/{id}/config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/config", req.URL.Path)
		var payload provisioning.ServiceConfigState
		json.NewDecoder(req.Body).Decode(&payload)
		r.Equal(configB, payload.ConfigID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(&payload)
	})
	// readServiceState at end of Update
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigB)
	})
	// Terraform Read (post-apply refresh)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigB)
	})
	// Terraform Read (verify no diff)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfigB)
	})

	// Destroy
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		w.WriteHeader(http.StatusAccepted)
	})
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&skysql.ErrorResponse{Code: http.StatusNotFound})
	})

	serviceHCL := func(cfgID string) string {
		return fmt.Sprintf(`
		resource "skysql_service" "default" {
			service_type        = "transactional"
			topology            = "es-single"
			cloud_provider      = "gcp"
			region              = "us-central1"
			name                = "test-swap-config"
			architecture        = "amd64"
			nodes               = 1
			size                = "sky-2x8"
			storage             = 100
			ssl_enabled         = true
			version             = "10.6.11-6-1"
			wait_for_creation   = true
			wait_for_deletion   = true
			deletion_protection = false
			config_id           = "%s"
		}`, cfgID)
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: serviceHCL(configA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_service.default", "config_id", configA),
				),
			},
			{
				Config: serviceHCL(configB),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_service.default", "config_id", configB),
				),
			},
		},
	})
}

// TestServiceResourceConfigID_RemoveConfig verifies that removing config_id
// reverts the service to the default config via DELETE /services/{id}/config.
func TestServiceResourceConfigID_RemoveConfig(t *testing.T) {
	configureOnce.Reset()

	const serviceID = "dbdgf42002423"
	const configID = "cfg-to-remove"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	service := &provisioning.Service{
		ID:           serviceID,
		Name:         "test-remove-config",
		Region:       "us-central1",
		Provider:     "gcp",
		Tier:         "power",
		Topology:     "es-single",
		Version:      "10.6.11-6-1",
		Architecture: "amd64",
		Size:         "sky-2x8",
		Nodes:        1,
		SSLEnabled:   true,
		Status:       "ready",
		CreatedOn:    int(time.Now().Unix()),
		UpdatedOn:    int(time.Now().Unix()),
		CreatedBy:    uuid.New().String(),
		UpdatedBy:    uuid.New().String(),
		Endpoints: []provisioning.Endpoint{
			{
				Name: "primary",
				Ports: []provisioning.Port{
					{Name: "readwrite", Port: 3306, Purpose: "readwrite"},
				},
			},
		},
		StorageVolume: struct {
			Size       int    `json:"size"`
			VolumeType string `json:"volume_type"`
			IOPS       int    `json:"iops"`
			Throughput int    `json:"throughput"`
		}{Size: 100, VolumeType: "pd-ssd"},
		IsActive:    true,
		ServiceType: "transactional",
	}

	serviceWithConfig := *service
	serviceWithConfig.ConfigID = configID

	// --- Step 1: Create service with config_id ---

	// Provider configure
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})
	// Create: POST /services
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		w.Header().Set("Content-Type", "application/json")
		pendingService := *service
		pendingService.Status = "pending_create"
		json.NewEncoder(w).Encode(&pendingService)
	})
	// Wait for creation
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// readServiceState after wait
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// Apply config: POST /services/{id}/config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/config", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(&provisioning.ServiceConfigState{ConfigID: configID})
	})
	// Wait for config apply
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})
	// Read after create
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})

	// --- Step 2: Remove config_id → revert to default ---

	// Terraform Read (pre-plan refresh)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})
	// Update: GET /services/{id} to check actual config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&serviceWithConfig)
	})
	// Remove config: DELETE /services/{id}/config
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID+"/config", req.URL.Path)
		w.WriteHeader(http.StatusAccepted)
	})
	// readServiceState at end of Update
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// Terraform Read (post-apply refresh)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})
	// Terraform Read (verify no diff)
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(service)
	})

	// Destroy
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		w.WriteHeader(http.StatusAccepted)
	})
	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(&skysql.ErrorResponse{Code: http.StatusNotFound})
	})

	serviceHCLWithConfig := fmt.Sprintf(`
	resource "skysql_service" "default" {
		service_type        = "transactional"
		topology            = "es-single"
		cloud_provider      = "gcp"
		region              = "us-central1"
		name                = "test-remove-config"
		architecture        = "amd64"
		nodes               = 1
		size                = "sky-2x8"
		storage             = 100
		ssl_enabled         = true
		version             = "10.6.11-6-1"
		wait_for_creation   = true
		wait_for_deletion   = true
		deletion_protection = false
		config_id           = "%s"
	}`, configID)

	serviceHCLNoConfig := `
	resource "skysql_service" "default" {
		service_type        = "transactional"
		topology            = "es-single"
		cloud_provider      = "gcp"
		region              = "us-central1"
		name                = "test-remove-config"
		architecture        = "amd64"
		nodes               = 1
		size                = "sky-2x8"
		storage             = 100
		ssl_enabled         = true
		version             = "10.6.11-6-1"
		wait_for_creation   = true
		wait_for_deletion   = true
		deletion_protection = false
	}`

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: serviceHCLWithConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_service.default", "config_id", configID),
				),
			},
			{
				Config: serviceHCLNoConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("skysql_service.default", "config_id"),
				),
			},
		},
	})
}
