package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	skysql "github.com/mariadb-corporation/skysql-api-go"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"api_key": {
					Type:        schema.TypeString,
					Required:    true,
					Optional:    false,
					DefaultFunc: schema.EnvDefaultFunc("TF_SKYSQL_API_KEY", nil),
					Sensitive:   true,
				},
				"mdbid_url": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("TF_SKYSQL_MDBID_URL", "https://id-dev.mariadb.com"),
				},
				"host": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("TF_SKYSQL_HOST", "https://api.dev.gcp.mariadb.net"),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"skysql_service": dataSourceService(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"skysql_service": resourceService(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var diags diag.Diagnostics

		apiKey := d.Get("api_key").(string)
		if apiKey == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "API Key not provided for SkySQL client",
				Detail:   "An API Key generated from MariaDB ID must be provided to authenticate",
			})
			return nil, diags
		}

		mdbid_url := d.Get("mdbid_url").(string)
		url, err := url.Parse(mdbid_url)
		if err != nil || url.String() == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to parse MariaDB ID url",
				Detail:   fmt.Sprintf("An invalid url was provided for MariaDB ID %s", err),
			})
			return nil, diags
		}

		url.Path = path.Join(url.Path, "/api/v1/token")
		req, _ := http.NewRequest("POST", url.String(), nil)
		req.Header.Add("Authorization", "Token "+apiKey)

		httpClient := &http.Client{}
		res, err := httpClient.Do(req)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate with MariaDB ID",
				Detail:   fmt.Sprintf("An invalid url was provided for MariaDB ID %s", err),
			})
			return nil, diags
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Unable to read response from MariaDB ID",
					Detail:   fmt.Sprintf("Failure occurred during authentication attempt %s", err),
				})
				return nil, diags
			}

			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate with MariaDB ID",
				Detail:   fmt.Sprintf("Failure occurred during authentication attempt. URL: %v, Status: %v, Body: %v", url, res.StatusCode, string(body)),
			})
			return nil, diags
		}

		var resData struct {
			Token string `json:"token"`
		}
		err = json.NewDecoder(res.Body).Decode(&resData)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate with MariaDB ID",
				Detail:   fmt.Sprintf("Failure to decode token response %s", err),
			})
			return nil, diags
		}

		slt := resData.Token
		if slt == "" {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate with MariaDB ID",
				Detail:   fmt.Sprintf("Token not in response from MariaDB ID. %v", resData),
			})
			return nil, diags
		}

		bearerTokenProvider, err := securityprovider.NewSecurityProviderBearerToken(slt)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to authenticate with MariaDB ID",
				Detail:   fmt.Sprintf("Failure to instantiate bearer token provider %s", err),
			})
			return nil, diags
		}

		userAgent := p.UserAgent("terraform-provider-skysql", version)
		client, err := skysql.NewClient(
			d.Get("host").(string),
			skysql.WithRequestEditorFn(bearerTokenProvider.Intercept),
			skysql.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
				req.Header.Set("User-Agent", userAgent)
				return nil
			}),
		)
		if err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Unable to create MariaDB ID client",
				Detail:   fmt.Sprintf("Failure to instantiate client %s", err),
			})
			return nil, diags
		}

		return client, nil
	}
}
