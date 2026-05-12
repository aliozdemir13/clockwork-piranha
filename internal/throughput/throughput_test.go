package throughput

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aliozdemir13/clockwork-piranha/internal/data_factory"
)

func TestThroughput_TestSuite(t *testing.T) {
	// Mock server to handle all factory requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	// 1. Test Success Path for TestSuite
	t.Run("TestSuite Success", func(t *testing.T) {
		tp := &Throughput{
			InstanceURL: server.URL,
			Ml: []*data_factory.MemberDetails{
				{
					ID:               "1",
					MembershipNumber: "M1",
					PersonAccountID:  "A1",
					Vouchers:         []data_factory.Voucher{{VoucherCode: "V1"}},
				},
			},
			Token:        "fake-token",
			TestRPS:      10,
			TestDuration: 10 * time.Millisecond, // Short duration for fast tests
		}

		// This calls runTest 5 times.
		// We expect files errors_GetVouchers.json, etc. to be created.
		tp.TestSuite()

		// Cleanup generated files
		files := []string{"errors_GetVouchers.json", "errors_ActivateVouchers.json", "errors_ValidateVouchers.json", "errors_GetConsent.json", "errors_SetConsent.json"}
		for _, f := range files {
			_ = os.Remove(f)
		}
	})

	// 2. Test Panic Path for TestSuite (Empty List)
	t.Run("TestSuite Panic on Empty List", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("TestSuite did not panic on empty member list")
			}
		}()
		tp := &Throughput{Ml: []*data_factory.MemberDetails{}}
		tp.TestSuite()
	})
}

func TestThroughput_runTest_Table(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the URL contains our "trigger" keyword
		if strings.Contains(r.URL.RawQuery, "triggerError=true") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"forced internal error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	members := []*data_factory.MemberDetails{
		{ID: "1", MembershipNumber: "M1", Vouchers: []data_factory.Voucher{{VoucherCode: "V1"}}},
	}

	tests := []struct {
		name         string
		testName     string
		status       int
		instanceURL  string
		rps          int
		duration     time.Duration
		triggerError bool
		emptyQueue   bool // To test the refill logic
	}{
		{
			name:         "Success Path",
			testName:     "SuccessTest",
			instanceURL:  server.URL,
			rps:          100,
			duration:     20 * time.Millisecond,
			triggerError: false,
		},
		{
			name:         "API Error Path (Status 500)",
			testName:     "ErrorTest",
			instanceURL:  server.URL,
			rps:          50,
			duration:     20 * time.Millisecond,
			triggerError: true,
		},
		{
			name:         "Network Error Path (Invalid URL)",
			testName:     "NetworkErr",
			instanceURL:  "http://localhost:1", // Connection refused
			rps:          10,
			duration:     10 * time.Millisecond,
			triggerError: true,
		},
		{
			name:        "Queue Refill Logic",
			testName:    "RefillTest",
			instanceURL: server.URL,
			rps:         200,
			duration:    30 * time.Millisecond, // High RPS + Duration forces multiple refills
			emptyQueue:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := &Throughput{
				InstanceURL:  tt.instanceURL,
				Ml:           members,
				Token:        "token",
				TestRPS:      tt.rps,
				TestDuration: tt.duration,
			}

			// server needs to responds with 500 when triggerError is true
			// Since it is not easy to change the factory per request inside runTest loop easily,
			// this alters server to send error.
			factory := func(m *data_factory.MemberDetails) *data_factory.TestCase {
				tc := GetVoucherFactory(tt.instanceURL)(m)
				if tt.triggerError {
					tc.Path += "&triggerError=true"
				}
				return tc
			}

			var queue []*data_factory.MemberDetails
			if !tt.emptyQueue {
				queue = append([]*data_factory.MemberDetails{}, members...)
			}

			// If testing error path, temporarily redirect server behavior
			tp.runTest(tt.testName, factory, queue)

			// Cleanup
			_ = os.Remove(fmt.Sprintf("errors_%s.json", tt.testName))
		})
	}

	// 3. Final Coverage Edge Case: os.WriteFile error
	t.Run("File Write Error", func(t *testing.T) {
		tp := &Throughput{
			InstanceURL:  server.URL,
			Ml:           members,
			TestRPS:      1,
			TestDuration: 5 * time.Millisecond,
		}
		// Passing a path that cannot be created (directory that doesn't exist)
		// triggers the "Failed to write JSON file" branch.
		tp.runTest("nonexistent_dir/test", GetVoucherFactory(server.URL), nil)
	})
}
