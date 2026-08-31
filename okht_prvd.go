package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"ton-module/internal/client" // ⚠️ adapte au vrai chemin du module (voir go.mod)
)

var _ provider.Provider = &monProvider{}

type monProvider struct {
	version string
}

type monProviderModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &monProvider{version: version}
	}
}

func (p *monProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mon_provider"
	resp.Version = p.version
}

func (p *monProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{Optional: true},
			"username": schema.StringAttribute{Optional: true},
			"password": schema.StringAttribute{Optional: true, Sensitive: true},
		},
	}
}

func (p *monProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config monProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := config.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = os.Getenv("MON_PROVIDER_BASE_URL")
	}
	username := config.Username.ValueString()
	if username == "" {
		username = os.Getenv("MON_PROVIDER_USERNAME")
	}
	password := config.Password.ValueString()
	if password == "" {
		password = os.Getenv("MON_PROVIDER_PASSWORD")
	}

	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(path.Root("base_url"), "base_url manquant", "Définis 'base_url' dans le bloc provider ou via MON_PROVIDER_BASE_URL.")
	}
	if username == "" {
		resp.Diagnostics.AddAttributeError(path.Root("username"), "username manquant", "Définis 'username' dans le bloc provider ou via MON_PROVIDER_USERNAME.")
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "password manquant", "Définis 'password' dans le bloc provider ou via MON_PROVIDER_PASSWORD.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// 👇 client. préfixe le type et la fonction, car importés depuis internal/client
	apiClient := client.NewAPIClient(baseURL)

	if err := apiClient.Login(username, password); err != nil {
		resp.Diagnostics.AddError("Échec de l'authentification auprès de l'API", "Impossible de récupérer le token bearer : "+err.Error())
		return
	}

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *monProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCurrentUserDataSource,
		NewLoginDataSource,
	}
}

func (p *monProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMonResource,
	}
}
