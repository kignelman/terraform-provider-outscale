package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type LoginRequest struct {
	Username       string `json:"username"`
	Base64Password string `json:"base64_password"`
}

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

// DoAuthenticatedRequest — en majuscule pour être exportée et utilisable depuis le package provider
func (c *APIClient) DoAuthenticatedRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)
}
