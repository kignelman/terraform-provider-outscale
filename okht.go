package main

import (
	"bytes"
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

// LoginRequest correspond au body attendu par l'endpoint d'auth
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse correspond à la réponse de l'endpoint d'auth
// (adapte le nom du champ selon le swagger, ex: "token", "access_token", "bearer_token")
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

// Login récupère le token bearer et le stocke dans le client
func (c *APIClient) Login(username, password string) error {
	payload := LoginRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erreur d'encodage du payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/auth/login", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("erreur de création de la requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("erreur lors de l'appel de login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login échoué, status %d: %s", resp.StatusCode, string(respBody))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("erreur de décodage de la réponse: %w", err)
	}

	c.Token = loginResp.Token
	return nil
}

// doAuthenticatedRequest ajoute automatiquement le header Authorization
func (c *APIClient) doAuthenticatedRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)
}

// Exemple : appel à une action de l'API protégée
func (c *APIClient) GetSomething() ([]byte, error) {
	resp, err := c.doAuthenticatedRequest(http.MethodGet, "/some/protected/endpoint", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("appel échoué, status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

func main() {
	client := NewAPIClient("https://api.example.com")

	if err := client.Login("mon_user", "mon_password"); err != nil {
		fmt.Println("Erreur de login:", err)
		return
	}
	fmt.Println("Token récupéré:", client.Token)

	data, err := client.GetSomething()
	if err != nil {
		fmt.Println("Erreur:", err)
		return
	}
	fmt.Println("Réponse:", string(data))
}
