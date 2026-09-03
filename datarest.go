package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"ton-module/internal/client" // ⚠️ adapte au chemin exact de ton go.mod
)

func NewRestsDataSource() datasource.DataSource {
	return &restsDataSource{}
}

type restsDataSource struct {
	client *client.APIClient
}

// --- Modèle Terraform ---

// restsDataSourceModel = le data source complet (contient la liste)
type restsDataSourceModel struct {
	Items []restItemModel `tfsdk:"items"`
}

// restItemModel = un élément de la liste
type restItemModel struct {
	Name  types.String `tfsdk:"name"`
	Rests types.String `tfsdk:"rests"`
}

// --- Struct brut pour décoder le JSON de l'API ---

type apiRestItem struct {
	Name  string `json:"name"`
	Rests string `json:"rests"`
}

func (d *restsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rests"
}

func (d *restsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Liste des éléments retournés par l'API",
		Attributes: map[string]schema.Attribute{
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},
						"rests": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *restsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *restsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// ⚠️ Path à adapter selon ton swagger
	httpResp, err := d.client.DoAuthenticatedRequest("GET", "/rests", nil)
	if err != nil {
		resp.Diagnostics.AddError("Erreur d'appel API", err.Error())
		return
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Erreur de lecture du body", err.Error())
		return
	}

	if httpResp.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Erreur API",
			fmt.Sprintf("status %d reçu : %s", httpResp.StatusCode, string(bodyBytes)),
		)
		return
	}

	// Décodage direct en slice (le JSON est un tableau à la racine)
	var apiItems []apiRestItem
	if err := json.Unmarshal(bodyBytes, &apiItems); err != nil {
		resp.Diagnostics.AddError("Erreur de décodage", err.Error())
		return
	}

	// Mapping : slice brut -> slice de modèles Terraform
	items := make([]restItemModel, 0, len(apiItems))
	for _, item := range apiItems {
		items = append(items, restItemModel{
			Name:  types.StringValue(item.Name),
			Rests: types.StringValue(item.Rests),
		})
	}

	state := restsDataSourceModel{
		Items: items,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
