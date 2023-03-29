package skysql

import (
	"context"
	"errors"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql/autonomous"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql/organization"
	"github.com/mariadb-corporation/terraform-provider-skysql/internal/skysql/provisioning"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

type Client struct {
	HTTPClient *resty.Client
}

func New(baseURL string, AccessToken string) *Client {
	transport := logging.NewLoggingHTTPTransport(http.DefaultTransport)

	clientName, _ := os.Executable()

	return &Client{
		HTTPClient: resty.NewWithClient(&http.Client{Transport: transport}).
			SetHeader("User-Agent", filepath.Base(clientName)).
			SetAuthScheme("Bearer").
			SetAuthToken(AccessToken).
			SetBaseURL(baseURL).
			EnableTrace(),
	}
}

func (c *Client) GetProjects(ctx context.Context) ([]organization.Project, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult([]organization.Project{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/organization/v1/projects")
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return *resp.Result().(*[]organization.Project), err
}

func WithPageSize(value uint) func(url.Values) {
	return func(values url.Values) {
		values.Set("page_size", strconv.Itoa(int(value)))
	}
}

func (c *Client) GetVersions(ctx context.Context, options ...func(url.Values)) ([]provisioning.Version, error) {
	request := c.HTTPClient.R()
	for _, option := range options {
		option(request.QueryParam)
	}
	resp, err := request.
		SetHeader("Accept", "application/json").
		SetResult([]provisioning.Version{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/provisioning/v1/versions")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}
	return *resp.Result().(*[]provisioning.Version), err
}

func (c *Client) GetServiceByID(ctx context.Context, serviceID string) (*provisioning.Service, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Service{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/provisioning/v1/services/" + serviceID)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return resp.Result().(*provisioning.Service), err
}

func (c *Client) CreateService(ctx context.Context, req *provisioning.CreateServiceRequest) (*provisioning.Service, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Service{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		SetBody(req).
		Post("/provisioning/v1/services")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}

	return resp.Result().(*provisioning.Service), err
}

func (c *Client) DeleteServiceByID(ctx context.Context, serviceID string) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Delete("/provisioning/v1/services/" + serviceID)
	if err != nil {
		return err
	}

	if resp.IsError() {
		return handleError(resp)
	}

	return nil
}

func (c *Client) GetServiceCredentialsByID(ctx context.Context, serviceID string) (*provisioning.Credentials, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.Credentials{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/provisioning/v1/services/" + serviceID + "/security/credentials")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	return resp.Result().(*provisioning.Credentials), err
}

func (c *Client) UpdateServiceAllowListByID(ctx context.Context, serviceID string, allowlist []provisioning.AllowListItem) ([]provisioning.AllowListItem, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetResult(provisioning.ReadAllowListResponse{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		SetBody(allowlist).
		Put("/provisioning/v1/services/" + serviceID + "/security/allowlist")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.ReadAllowListResponse)

	return response[0].AllowList, err
}

func (c *Client) ReadServiceAllowListByID(ctx context.Context, serviceID string) (provisioning.ReadAllowListResponse, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetResult(provisioning.ReadAllowListResponse{}).
		SetError(&ErrorResponse{}).
		Get("/provisioning/v1/services/" + serviceID + "/security/allowlist")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.ReadAllowListResponse)
	if response == nil {
		response = make(provisioning.ReadAllowListResponse, 0)
	}
	return response, err
}

func handleError(resp *resty.Response) error {
	if resp.StatusCode() == 404 {
		return ErrorServiceNotFound
	}
	if resp.StatusCode() == 401 {
		return ErrorUnauthorized
	}
	if resp.Error() != nil {
		if resp.StatusCode() == 500 {
			return errors.New("SkySQL API returned 500 Internal Server Error")
		}
		errResp := resp.Error().(*ErrorResponse)
		return errors.New(errResp.Errors[0].Message)
	}
	return errors.New(resp.Status())
}

func (c *Client) SetServicePowerState(ctx context.Context, serviceID string, isActive bool) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.PowerStateRequest{IsActive: isActive}).
		SetError(&ErrorResponse{}).
		Post("/provisioning/v1/services/" + serviceID + "/power")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) ModifyServiceEndpoints(
	ctx context.Context,
	serviceID string,
	mechanism string,
	allowedAccounts []string,
	visibility string,
) (*provisioning.ServiceEndpoint, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.PatchServiceEndpointsRequest{
			{Mechanism: mechanism,
				AllowedAccounts: allowedAccounts,
				Visibility:      visibility},
		}).
		SetResult(provisioning.PatchServiceEndpointsResponse{}).
		SetError(&ErrorResponse{}).
		Patch("/provisioning/v1/services/" + serviceID + "/endpoints")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}
	response := *resp.Result().(*provisioning.PatchServiceEndpointsResponse)
	if response == nil {
		response = make(provisioning.PatchServiceEndpointsResponse, 0)
	}
	return &response[0], err
}

func (c *Client) ModifyServiceSize(ctx context.Context, serviceID string, size string) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateServiceSizeRequest{Size: size}).
		SetError(&ErrorResponse{}).
		Post("/provisioning/v1/services/" + serviceID + "/size")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) ModifyServiceNodeNumber(ctx context.Context, serviceID string, nodes int64) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateServiceNodesNumberRequest{Nodes: nodes}).
		SetError(&ErrorResponse{}).
		Post("/provisioning/v1/services/" + serviceID + "/nodes")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) ModifyServiceStorage(ctx context.Context, serviceID string, size int64, iops int64) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(&provisioning.UpdateStorageRequest{Size: size, IOPS: iops}).
		SetError(&ErrorResponse{}).
		Patch("/provisioning/v1/services/" + serviceID + "/storage")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) SetAutonomousActions(
	ctx context.Context,
	value autonomous.SetAutonomousActionsRequest,
) ([]autonomous.ActionResponse, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetBody(value).
		SetResult([]autonomous.ActionResponse{}).
		SetError(&ErrorResponse{}).
		Post("/als/v1/actions")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}

	response := *resp.Result().(*[]autonomous.ActionResponse)
	if response == nil {
		response = make([]autonomous.ActionResponse, 0)
	}
	return response, err
}

func (c *Client) GetAutonomousActions(ctx context.Context, serviceID string) ([]autonomous.ActionResponse, error) {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetResult([]autonomous.ActionResponse{}).
		SetError(&ErrorResponse{}).
		SetQueryParam("service_id", serviceID).
		Get("/als/v1/actions")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, handleError(resp)
	}

	response := *resp.Result().(*[]autonomous.ActionResponse)
	if response == nil {
		response = make([]autonomous.ActionResponse, 0)
	}
	return response, err
}

func (c *Client) DeleteAutonomousAction(ctx context.Context, actionID string) error {
	resp, err := c.HTTPClient.R().
		SetHeader("Accept", "application/json").
		SetContext(ctx).
		SetError(&ErrorResponse{}).
		Delete("/als/v1/actions/" + actionID)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return handleError(resp)
	}

	return err
}

func (c *Client) GetAvailabilityZones(ctx context.Context, region string, options ...func(url.Values)) ([]provisioning.AvailabilityZone, error) {
	request := c.HTTPClient.R()
	for _, option := range options {
		option(request.QueryParam)
	}
	resp, err := request.
		SetHeader("Accept", "application/json").
		SetResult([]provisioning.AvailabilityZone{}).
		SetError(&ErrorResponse{}).
		SetContext(ctx).
		Get("/provisioning/v1/regions/" + region + "/zones")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, handleError(resp)
	}
	return *resp.Result().(*[]provisioning.AvailabilityZone), err
}
