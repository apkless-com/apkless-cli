package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// apiRequest makes a request to the APKless API
func apiRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, apiBase+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+getAPIKey())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

// serverRequest makes a request to the phone's API server
func serverRequest(serverURL, token, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, serverURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// getPhoneConnection gets server_url and server_token for a phone
func getPhoneConnection(phoneID string) (serverURL, token string, err error) {
	result, err := apiRequest("GET", "/v1/phones/"+phoneID, nil)
	if err != nil {
		return "", "", err
	}

	status, _ := result["status"].(string)
	if status != "ready" {
		return "", "", fmt.Errorf("phone %s is not ready (status: %s)", phoneID, status)
	}

	serverURL, _ = result["server_url"].(string)
	token, _ = result["server_token"].(string)

	if serverURL == "" {
		return "", "", fmt.Errorf("phone %s has no server URL", phoneID)
	}
	return serverURL, token, nil
}

// printJSON pretty prints JSON
func printJSON(data []byte) {
	var v interface{}
	if json.Unmarshal(data, &v) == nil {
		b, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Println(string(data))
	}
}

