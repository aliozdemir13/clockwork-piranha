// Package main is the main executable
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aliozdemir13/clockwork-piranha/internal/data_factory"
	"github.com/aliozdemir13/clockwork-piranha/internal/throughput"
)

func main() {
	// read the API configurations
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter clientId: ")
	scanner.Scan()
	ClientID := scanner.Text()

	fmt.Println("Enter clientSecret: ")
	scanner.Scan()
	ClientSecret := scanner.Text()

	fmt.Println("Enter orgBaseUrl (https://domainName.my.salesforce.com/): ")
	scanner.Scan()
	InstanceURL := scanner.Text()

	fmt.Println("Enter how many request to be sent per minute: ")
	scanner.Scan()
	TestRPS, err := strconv.Atoi(scanner.Text())
	if err != nil {
		panic(fmt.Sprintf("Error parsing RPS : %v", err))
	}

	fmt.Println("Duration (Mins): ")
	scanner.Scan()
	TestDurationInput, err := strconv.Atoi(scanner.Text())
	if err != nil {
		panic(fmt.Sprintf("Error parsing RPS : %v", err))
	}

	fmt.Println("----- Throughput/Concurrency Test Starts -------")

	// test duration calculation
	TestDuration := time.Duration(TestDurationInput) * time.Minute

	// read member list, also to be used as cache
	// TODO: add a logic to generate pool of data, current version only uses CSV file
	ml, err := data_factory.LoadMembersFromCSV("membersFromFullSBX.csv")
	if err != nil {
		panic(fmt.Sprintf("Something went wrong during data load: %s", err))
	}

	// print first x samples, auditability only
	i := 0
	for _, m := range ml {
		if i < 15 {
			fmt.Printf("%s %s %s %v", m.ID, m.MembershipNumber, m.PersonAccountID, m.Vouchers)
			i++
		}
	}

	token, err := getSalesforceToken(ClientID, ClientSecret, InstanceURL)
	if err != nil {
		panic(fmt.Sprintf("Auth failed: %v", err))
	}
	fmt.Printf("Authenticated successfully. Starting tests...\n\n")

	tp := &throughput.Throughput{
		InstanceURL:  InstanceURL,
		Ml:           ml,
		Token:        token,
		TestRPS:      TestRPS,
		TestDuration: TestDuration,
	}

	tp.TestSuite()
}

// getSalesforceToken handles authentication to org with the given user credentials context
func getSalesforceToken(ClientID string, ClientSecret string, InstanceURL string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)

	resp, err := http.PostForm(InstanceURL+"/services/oauth2/token", data)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var auth data_factory.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}
	return auth.AccessToken, nil
}
