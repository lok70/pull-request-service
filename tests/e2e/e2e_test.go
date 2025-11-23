package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8090"

func TestE2E_FullFlow(t *testing.T) {
	waitForService(t)

	client := &http.Client{Timeout: 5 * time.Second}

	t.Log("Step 1: Create Team with Users")
	teamBody := []byte(`{
		"team_name": "gophers_e2e",
		"members": [
			{"user_id": "u1", "username": "Alice", "is_active": true},
			{"user_id": "u2", "username": "Bob", "is_active": true},
			{"user_id": "u3", "username": "Charlie", "is_active": true}
		]
	}`)

	resp, err := client.Post(baseURL+"/team/add", "application/json", bytes.NewBuffer(teamBody))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 1 Failed: Expected 200/201, got %d", resp.StatusCode)
	}
	t.Log("Step 1: Success")

	// --- ШАГ 2: Создание PR ---
	t.Log("Step 2: Create Pull Request")
	prBody := []byte(`{
		"pull_request_id": "pr-100",
		"pull_request_name": "Feature Login",
		"author_id": "u1"
	}`)

	resp, err = client.Post(baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(prBody))
	if err != nil {
		t.Fatalf("Failed to create PR: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Step 2 Failed: Expected 201, got %d", resp.StatusCode)
	}

	var prResp struct {
		PR struct {
			ID        string   `json:"pull_request_id"`
			Reviewers []string `json:"assigned_reviewers"`
			Status    string   `json:"status"`
		} `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
		t.Fatal("Failed to decode PR response:", err)
	}

	if prResp.PR.Status != "OPEN" {
		t.Errorf("Expected status OPEN, got %s", prResp.PR.Status)
	}
	if len(prResp.PR.Reviewers) != 2 {
		t.Errorf("Expected 2 reviewers, got %d", len(prResp.PR.Reviewers))
	}
	t.Logf("Step 2 Success: PR created with reviewers: %v", prResp.PR.Reviewers)

	t.Log("Step 3: Check Stats")
	resp, err = client.Get(baseURL + "/stats")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Step 3 Failed: Expected 200, got %d", resp.StatusCode)
	}

	var stats []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal("Failed to decode stats:", err)
	}
	if len(stats) == 0 {
		t.Error("Expected stats to be not empty")
	}
	t.Log("Step 3: Success")

	t.Log("Step 4: Merge PR")
	mergeBody := []byte(`{
		"pull_request_id": "pr-100"
	}`)

	resp, err = client.Post(baseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer(mergeBody))
	if err != nil {
		t.Fatalf("Failed to merge PR: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Step 4 Failed: Expected 200, got %d", resp.StatusCode)
	}

	// Проверяем, что статус сменился на MERGED
	var mergeResp struct {
		PR struct {
			Status string `json:"status"`
		} `json:"pr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mergeResp); err != nil {
		t.Fatal("Failed to decode Merge response:", err)
	}

	if mergeResp.PR.Status != "MERGED" {
		t.Errorf("Expected status MERGED, got %s", mergeResp.PR.Status)
	}
	t.Log("Step 4: Success (PR Merged)")

	t.Log("Step 4.1: Check Idempotency")
	resp, err = client.Post(baseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer(mergeBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Idempotency check failed: Expected 200 on second call, got %d", resp.StatusCode)
	}
	t.Log("Step 4.1: Success")

	t.Log("Step 5: Deactivate User")
	activeBody := []byte(`{
		"user_id": "u1",
		"is_active": false
	}`)

	resp, err = client.Post(baseURL+"/users/setIsActive", "application/json", bytes.NewBuffer(activeBody))
	if err != nil {
		t.Fatalf("Failed to deactivate user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Step 5 Failed: Expected 200, got %d", resp.StatusCode)
	}

	var userResp struct {
		User struct {
			UserID   string `json:"user_id"`
			IsActive bool   `json:"is_active"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		t.Fatal("Failed to decode User response:", err)
	}

	if userResp.User.IsActive != false {
		t.Errorf("Expected is_active=false, got true")
	}
	t.Log("Step 5: Success (User Deactivated)")
}

func waitForService(t *testing.T) {
	t.Log("Waiting for service to start...")
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Service did not start in time")
		case <-ticker.C:
			resp, err := http.Get(baseURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				t.Log("Service is UP!")
				return
			}
		}
	}
}
