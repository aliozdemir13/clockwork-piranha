// Package data_factory handles data preparation and randomization
package data_factory

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// LoadMembersFromCSV handles the test pool generation
func LoadMembersFromCSV(filename string) ([]*MemberDetails, error) {
	// TODO: dynamic voucher pool creation
	vouchersPool := []Voucher{
		{VoucherCode: "12345678"},
		{VoucherCode: "98765432"},
		{VoucherCode: "12743456"},
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	reader := csv.NewReader(f)
	// precaution, for all the salesforce reads I have done, quotes caused the issue
	reader.LazyQuotes = true
	// Optional safety: If some lines are shorter than others, don't crash
	reader.FieldsPerRecord = -1

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV Error: %v", err)
	}

	var members []*MemberDetails
	for i, line := range lines {
		if i == 0 {
			continue // skip header
		}

		// SAFETY CHECK: Ensure the line actually has the 3 columns needed
		if len(line) < 3 {
			continue
		}

		members = append(members, &MemberDetails{
			// We use strings.Trim to remove any quotes that LazyQuotes might have left behind
			ID:               strings.Trim(line[0], "\""),
			MembershipNumber: strings.Trim(line[1], "\""),
			PersonAccountID:  strings.Trim(line[2], "\""),
			Vouchers:         vouchersPool,
		})
	}
	return members, nil
}
