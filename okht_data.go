package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"ton-module/internal/client" // ⚠️ adapte au chemin exact de ton go.mod
)

// NewLoginDataSource retourne une nouvelle instance du data source
func NewLoginDataSource() datasource.DataSource {
	return &loginDataSource{}
}

type loginDataSource struct {
	client *client.APIClient // injecté via Configure, déjà authentifié
}

// loginDataSourceModel correspond au schema exposé côté Terraform
// (adapte les champs selon ce que renvoie réellement ton endpoint utilisateur)
type loginDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
}

// apiUserResponse reflète exactement le JSON renvoyé par l'API (swagger)
type apiUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (d *loginDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_login"
}

func (d *loginDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Informations sur l'utilisateur actuellement authentifié",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"username": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Configure récupère le client déjà authentifié, injecté depuis provider.go
func (d *loginDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Type de données du provider inattendu",
			fmt.Sprintf("Attendu *client.APIClient, reçu %T", req.ProviderData),
		)
		return
	}
	d.client = apiClient
}

// Read appelle l'API avec le token bearer déjà présent dans d.client
func (d *loginDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// ⚠️ Path à adapter selon ton swagger (ex: /me, /user, /account, /profile)
	httpResp, err := d.client.DoAuthenticatedRequest("GET", "/me", nil)
	if err != nil {
		resp.Diagnostics.AddError("Erreur d'appel API", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Erreur API",
			fmt.Sprintf("status %d reçu lors de la récupération de l'utilisateur connecté", httpResp.StatusCode),
		)
		return
	}

	var apiUser apiUserResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiUser); err != nil {
		resp.Diagnostics.AddError("Erreur de décodage", err.Error())
		return
	}

	state := loginDataSourceModel{
		ID:       types.StringValue(apiUser.ID),
		Username: types.StringValue(apiUser.Username),
		Email:    types.StringValue(apiUser.Email),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
