package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient encapsule l'URL de base, le token et le client HTTP
type APIClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// LoginRequest correspond au body JSON attendu par l'endpoint de login
type LoginRequest struct {
	Username       string `json:"username"`
	Base64Password string `json:"base64_password"`
}

// LoginResponse correspond à la réponse de l'endpoint de login
// (adapte "token" si le champ s'appelle autrement dans ton swagger)
type LoginResponse struct {
	Token string `json:"token"`
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Login récupère le token bearer et le stocke dans c.Token
func (c *APIClient) Login(username, password string) error {
	payload := LoginRequest{
		Username:       username,
		Base64Password: base64.StdEncoding.EncodeToString([]byte(password)),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erreur d'encodage du payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/login", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("erreur de création de la requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("erreur lors de l'appel de login: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erreur de lecture du body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login échoué, status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(bodyBytes, &loginResp); err != nil {
		return fmt.Errorf("erreur de décodage de la réponse: %w", err)
	}

	c.Token = loginResp.Token
	return nil
}

// doAuthenticatedRequest ajoute automatiquement le header Authorization: Bearer
func (c *APIClient) doAuthenticatedRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)

}






func (d *currentUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
