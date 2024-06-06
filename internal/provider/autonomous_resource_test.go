package provider

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/autonomous"
	"github.com/skysqlinc/terraform-provider-skysql/internal/skysql/provisioning"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"regexp"
	"testing"
)

func TestAutonomousResource(t *testing.T) {
	const serviceID = "dbdgf42002418"
	const serviceName = "test-service"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		r.Equal("page_size=1", req.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})

	tests := []struct {
		name         string
		testResource string
		before       func(r *require.Assertions)
		checks       []resource.TestCheckFunc
		expectError  *regexp.Regexp
	}{
		{
			name: "create with empty actions",
			testResource: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
				}`, serviceID, serviceName),
			before: func(r *require.Assertions) {
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(getAutonomousByServiceID(t, serviceID))
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
			},
		},
		{
			name: "create with auto_scale_disk",
			testResource: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
		           auto_scale_disk = {
		           	enabled = true
						max_storage_size_gbs = 200
		           }
				}`, serviceID, serviceName),
			before: func(r *require.Assertions) {
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(setAutonomousResponse(t))
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(getAutonomousByServiceID(t, serviceID))
				expectRequest(deleteActionSuccessResponse(t))
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
			},
		},
		{
			name: "create with auto_scale_nodes_horizontal",
			testResource: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
		           auto_scale_nodes_horizontal = {
					  max_nodes = 2
					  min_nodes = 1
					}
				}`, serviceID, serviceName),
			before: func(r *require.Assertions) {
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(setAutonomousResponse(t))
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(getAutonomousByServiceID(t, serviceID))
				expectRequest(deleteActionSuccessResponse(t))
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
			},
		},
		{
			name: "create with auto_scale_nodes_vertical",
			testResource: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
		           auto_scale_nodes_vertical = {
					  min_size = "sky-2x8"
					  max_size = "sky-4x16"
					}
				}`, serviceID, serviceName),
			before: func(r *require.Assertions) {
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(setAutonomousResponse(t))
				expectRequest(getServiceByIDSuccess(t, serviceID))
				expectRequest(getAutonomousByServiceID(t, serviceID))
				expectRequest(deleteActionSuccessResponse(t))
			},
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
				resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
			},
		},
	}
	for _, test := range tests {
		{
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
}

func deleteActionSuccessResponse(t *testing.T) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodDelete, req.Method)
		r.Regexp(regexp.MustCompile("/als/v1/actions/[0-9a-f-]+"), req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

func TestModifyAutonomousResource(t *testing.T) {
	configureOnce.Reset()

	const serviceID = "dbdgf42002419"
	const serviceName = "test-service"

	testUrl, expectRequest, close := mockSkySQLAPI(t)
	defer close()
	os.Setenv("TF_SKYSQL_API_KEY", "[api-key]")
	os.Setenv("TF_SKYSQL_API_BASE_URL", testUrl)

	expectRequest(func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method, fmt.Sprintf("unexpected request %s %s", req.Method, req.URL.Path))
		r.Equal("/provisioning/v1/versions", req.URL.Path)
		r.Equal("page_size=1", req.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]provisioning.Version{})
	})
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(setAutonomousResponse(t))
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(getAutonomousByServiceID(t, serviceID))
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(getAutonomousByServiceID(t, serviceID))
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(deleteActionSuccessResponse(t))
	expectRequest(setAutonomousResponse(t))
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(getAutonomousByServiceID(t, serviceID))
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(getAutonomousByServiceID(t, serviceID))
	expectRequest(getServiceByIDSuccess(t, serviceID))
	for i := 0; i < 2; i++ {
		expectRequest(deleteActionSuccessResponse(t))
	}
	expectRequest(getServiceByIDSuccess(t, serviceID))
	expectRequest(getAutonomousByServiceID(t, serviceID))

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"skysql": providerserver.NewProtocol6WithError(New("")()),
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
                    auto_scale_disk = {
						max_storage_size_gbs = 200
                    }
                    auto_scale_nodes_horizontal = {
					  max_nodes = 2
					  min_nodes = 1
					}
				}`, serviceID, serviceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
			       auto_scale_disk = {
						max_storage_size_gbs = 300
			       }
			      auto_scale_nodes_vertical = {
					  min_node_size = "sky-2x8"
					  max_node_size = "sky-4x32"
					}
				}`, serviceID, serviceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "skysql_autonomous" "default" {
					service_id = "%s"
					service_name = "%s"
				}`, serviceID, serviceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("skysql_autonomous.default", "id", serviceID),
					resource.TestCheckResourceAttr("skysql_autonomous.default", "service_name", serviceName),
				),
			},
		},
	})
}

func getAutonomousByServiceID(t *testing.T, serviceID string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(
			fmt.Sprintf("%s %s", http.MethodGet, "/als/v1/actions"),
			fmt.Sprintf("%s %s", req.Method, req.URL.Path))
		r.Equal("service_id="+serviceID, req.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]autonomous.ActionResponse{})
	}
}

var groupToID = map[string]string{
	autonomous.AutoScaleDiskActionGroup:            uuid.New().String(),
	autonomous.AutoScaleNodesHorizontalActionGroup: uuid.New().String(),
	autonomous.AutoScaleNodesVerticalActionGroup:   uuid.New().String(),
}

func setAutonomousResponse(t *testing.T) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodPost, req.Method, fmt.Sprintf("unexpected request %s %s", req.Method, req.URL.Path))
		r.Equal("/als/v1/actions", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		payload := &autonomous.SetAutonomousActionsRequest{}
		err := json.NewDecoder(req.Body).Decode(payload)
		r.NoError(err)

		var response []autonomous.ActionResponse
		for _, action := range payload.Actions {
			response = append(response, autonomous.ActionResponse{
				Group:   action.Group,
				Enabled: action.Enabled,
				Params:  action.Params,
				ID:      groupToID[action.Group],
			})
		}

		json.NewEncoder(w).Encode(&response)
		r.NoError(err)

		w.WriteHeader(http.StatusOK)
	}
}

func getServiceByIDSuccess(t *testing.T, serviceID string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		r := require.New(t)
		r.Equal(http.MethodGet, req.Method)
		r.Equal("/provisioning/v1/services/"+serviceID, req.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		service := provisioning.Service{
			ID: serviceID,
		}
		json.NewEncoder(w).Encode(service)
	}
}
