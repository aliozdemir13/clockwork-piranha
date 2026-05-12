// Package throughput contains the test case factories for throughput tests and Execution logic. Each factory function returns a function that takes in MemberDetails and returns a TestCase struct with the appropriate API endpoint, method, and body for the test case.
package throughput

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aliozdemir13/clockwork-piranha/internal/data_factory"

	"golang.org/x/time/rate"
)

// Throughput struct holds the configuration for the throughput tests, including the Salesforce instance URL, the list of MemberDetails to use for generating test cases, the authentication token, desired RPS, and test duration.
type Throughput struct {
	InstanceURL  string
	Ml           []*data_factory.MemberDetails
	Token        string
	TestRPS      int
	TestDuration time.Duration
}

// TestSuite runs all the throughput test cases sequentially. Each test case is executed with its own randomized data queue, and results are printed at the end of each test case.
func (t *Throughput) TestSuite() {
	// getVouchers test

	// randomized data queue for actual test, once it is empty
	// original ml state will be used to reshuffle and refill
	mq, err := data_factory.ShuffleOnInit(t.Ml)
	if err != nil {
		panic(err)
	}
	vouchersFactory := GetVoucherFactory(t.InstanceURL)
	t.runTest("GetVouchers", vouchersFactory, mq)

	// activateDeactivateVouchers test
	mq, err = data_factory.ShuffleOnInit(t.Ml)
	if err != nil {
		panic(err)
	}
	activateFactory := ActivateVoucherFactory(t.InstanceURL)
	t.runTest("ActivateVouchers", activateFactory, mq)

	// validateVouchers test
	mq, err = data_factory.ShuffleOnInit(t.Ml)
	if err != nil {
		panic(err)
	}
	validateFactory := ValidateVoucherFactory(t.InstanceURL)
	t.runTest("ValidateVouchers", validateFactory, mq)

	// getConsent test
	mq, err = data_factory.ShuffleOnInit(t.Ml)
	if err != nil {
		panic(err)
	}
	getConsentFactory := GetConsentFactory(t.InstanceURL)
	t.runTest("GetConsent", getConsentFactory, mq)

	// setConsent test
	mq, err = data_factory.ShuffleOnInit(t.Ml)
	if err != nil {
		panic(err)
	}
	setConsentFactory := SetConsentFactory(t.InstanceURL)
	t.runTest("SetConsent", setConsentFactory, mq)
}

// runTest executes a single test case using the provided factory function to generate test cases from the MemberDetails queue. 
// It uses a rate limiter to control the RPS and a context to control the duration of the test. 
// Results are printed at the end of the test, including success count, error count, actual RPS, and top distinct error messages.
func (t *Throughput) runTest(TestName string, factory func(*data_factory.MemberDetails) *data_factory.TestCase, q []*data_factory.MemberDetails) {
	fmt.Printf(">>> Testing: %s (%d RPS for %v)\n", TestName, t.TestRPS, t.TestDuration)

	tr := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	limiter := rate.NewLimiter(rate.Limit(t.TestRPS), 1)

	// The context is what truly controls the duration
	ctx, cancel := context.WithTimeout(context.Background(), t.TestDuration)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount, errorCount int64
	uniqueErrors := make(map[string]int)

	startTime := time.Now()

	// Infinite loop, so only ctx (duration) can stop the process.
	// this is set up for avoiding end of line for data
	for i := 0; ; i++ {
		if err := limiter.Wait(ctx); err != nil {
			// This triggers when TestDuration is reached
			break
		}

		// Use the factory to get the data for ONLY this specific request
		var currentCase *data_factory.TestCase
		if len(q) > 0 {
			currentCase = factory(q[0]) // Get first element
			q = q[1:]                   // Remove it
		} else {
			// with complexity of O(N), this time will be negligble to refill the slice
			// Important: it will reset the the work done using the cache for the execution, but remote will still
			// keep the work done for the write operations, this test currently assumes that data pool is large
			// enough to cover the test and this is an error handling case mainly for read and to avoid silent failures
			// TODO: find a better solution to cover the write operation as well without breaking the report accuracy
			// double buffer perhaps or continuous feed
			// Edge case for continuous-feed mode: same record randomly selected twice in close succession. Statistically rare at our pool sizes, ignored.
			fresh, _ := data_factory.ShuffleOnInit(append([]*data_factory.MemberDetails(nil), t.Ml...))
			q = fresh
			if len(q) > 0 {
				currentCase = factory(q[0])
				q = q[1:]
			}
		}

		wg.Add(1)
		go func(tc *data_factory.TestCase) {
			defer wg.Done()

			req, err := http.NewRequest(tc.Method, tc.Path, bytes.NewBufferString(tc.Body))
			if err != nil {
				return
			}

			req.Header.Add("Authorization", "Bearer "+t.Token)
			req.Header.Add("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				errorCount++
				uniqueErrors[fmt.Sprintf("Network Error: %v", err)]++
				mu.Unlock()
				return
			}

			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			mu.Lock()
			if resp.StatusCode >= 400 {
				errorCount++
				// Log the status code + the error message from Salesforce
				errMsg := fmt.Sprintf("Status %d: %s", resp.StatusCode, string(body))
				uniqueErrors[errMsg]++
			} else {
				successCount++
			}
			mu.Unlock()

		}(currentCase) // Pass the captured case into the goroutine
	}

	wg.Wait()
	elapsed := time.Since(startTime).Seconds()

	fmt.Printf("Results: Success: %d | Errors: %d | Actual RPS: %.2f\n",
		successCount, errorCount, float64(successCount+errorCount)/elapsed)

	if len(uniqueErrors) > 0 {
		fmt.Println("Top Distinct Error Messages (Max 10):")
		count := 0
		for msg, freq := range uniqueErrors {
			if count >= 10 {
				break
			}
			fmt.Printf("[%d occurrences]: %s\n", freq, msg)
			count++
		}
	}

	fileName := fmt.Sprintf("errors_%s.json", TestName)
	fileData, _ := json.MarshalIndent(uniqueErrors, "", "  ")
	err := os.WriteFile(fileName, fileData, 0644)
	if err != nil {
		fmt.Printf("Failed to write JSON file: %v\n", err)
	} else {
		fmt.Printf("Full error log saved to: %s\n", fileName)
	}
	fmt.Println("--------------------------------------------------")
}
