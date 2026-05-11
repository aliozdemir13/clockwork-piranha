package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetSalesforceToken(t *testing.T) {
	tests := []struct {
		name         string
		mockStatus   int
		mockResponse string
		wantErr      bool
	}{
		{
			name:         "Success",
			mockStatus:   http.StatusOK,
			mockResponse: `{"access_token": "valid-token"}`,
			wantErr:      false,
		},
		{
			name:         "API Error 401",
			mockStatus:   http.StatusUnauthorized,
			mockResponse: `Unauthorized`,
			wantErr:      true,
		},
		{
			name:         "Invalid JSON",
			mockStatus:   http.StatusOK,
			mockResponse: `not-json`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				_, _ = w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			token, err := getSalesforceToken("id", "secret", server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSalesforceToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && token != "valid-token" {
				t.Errorf("Expected token valid-token, got %s", token)
			}
		})
	}

	t.Run("Network Error", func(t *testing.T) {
		_, err := getSalesforceToken("id", "secret", "http://0.0.0.0:0")
		if err == nil {
			t.Error("Expected network error, got nil")
		}
	})
}
func TestMainFunc(t *testing.T) {
	// Prepare a dummy CSV file that main() expects
	csvData := "id,membership_number,person_account_id\n1,M1,A1"
	err := os.WriteFile("membersFromFullSBX.csv", []byte(csvData), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("membersFromFullSBX.csv")

	// Mock Salesforce server for the auth call and the test suite
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "test-token"}`))
	}))
	defer server.Close()

	tests := []struct {
		name      string
		input     string
		wantPanic bool
		preTest   func() // Optional setup per test
		postTest  func() // Optional cleanup per test
	}{
		{
			name: "Successful Run",
			// clientId, secret, url, RPS, Duration
			input:     fmt.Sprintf("id\nsecret\n%s\n10\n0\n", server.URL),
			wantPanic: false,
		},
		{
			name:      "Invalid RPS Panic",
			input:     "id\nsecret\nurl\nNOT_A_NUMBER\n",
			wantPanic: true,
		},
		{
			name:      "Invalid Duration Panic",
			input:     "id\nsecret\nurl\n10\nNOT_A_NUMBER\n",
			wantPanic: true,
		},
		{
			name:      "Auth Failure Panic",
			input:     "id\nsecret\nhttp://invalid-url\n10\n0\n",
			wantPanic: true,
		},
		{
			name:    "Missing CSV Error (Now Panics)",
			input:   fmt.Sprintf("id\nsecret\n%s\n10\n0\n", server.URL),
			preTest: func() { os.Remove("membersFromFullSBX.csv") },
			postTest: func() {
				_ = os.WriteFile("membersFromFullSBX.csv", []byte("id,membership_number,person_account_id\n1,M1,A1"), 0644)
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preTest != nil {
				tt.preTest()
			}

			// Mock Stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			go func() {
				_, _ = io.WriteString(w, tt.input)
				_ = w.Close()
			}()

			// Handle Panic
			defer func() {
				os.Stdin = oldStdin
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("main() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
				if tt.postTest != nil {
					tt.postTest()
				}
			}()

			main()
		})
	}
}
