package throughput

import (
	"strings"
	"testing"

	"github.com/aliozdemir13/clockwork-piranha/internal/data_factory"
)

func TestFactories(t *testing.T) {
	instanceURL := "https://test.salesforce.com"
	member := &data_factory.MemberDetails{
		Id:               "rec123",
		MembershipNumber: "MEM-999",
		PersonAccountId:  "ACC-000",
		Vouchers: []data_factory.Voucher{
			{VoucherCode: "VOUCH1"},
			{VoucherCode: "VOUCH2"},
		},
	}

	tests := []struct {
		name           string
		factoryFunc    func(string) func(*data_factory.MemberDetails) *data_factory.TestCase
		expectedName   string
		expectedMethod string
		// Validation logic for the specific factory output
		validate func(t *testing.T, tc *data_factory.TestCase)
	}{
		{
			name:           "GetVoucherFactory",
			factoryFunc:    GetVoucherFactory,
			expectedName:   "getVouchers",
			expectedMethod: "GET",
			validate: func(t *testing.T, tc *data_factory.TestCase) {
				if !strings.Contains(tc.Path, member.MembershipNumber) {
					t.Errorf("Path missing membership number: %s", tc.Path)
				}
				if !strings.Contains(tc.Path, "language=de") {
					t.Error("Path missing language param")
				}
			},
		},
		{
			name:           "ActivateVoucherFactory",
			factoryFunc:    ActivateVoucherFactory,
			expectedName:   "activateVoucher",
			expectedMethod: "POST",
			validate: func(t *testing.T, tc *data_factory.TestCase) {
				if !strings.Contains(tc.Body, member.MembershipNumber) {
					t.Error("Body missing membership number")
				}
				if !strings.Contains(tc.Body, "isActive") {
					t.Error("Body missing IsActive field")
				}
				// Verify one of the random vouchers was picked
				if !strings.Contains(tc.Body, "VOUCH1") && !strings.Contains(tc.Body, "VOUCH2") {
					t.Error("Body missing valid VoucherCode from pool")
				}
			},
		},
		{
			name:           "ValidateVoucherFactory",
			factoryFunc:    ValidateVoucherFactory,
			expectedName:   "validateVouchers",
			expectedMethod: "POST",
			validate: func(t *testing.T, tc *data_factory.TestCase) {
				if !strings.Contains(tc.Body, member.MembershipNumber) {
					t.Error("Body missing membership number")
				}
				if !strings.Contains(tc.Path, "ValidateVoucherRedemption") {
					t.Error("Path incorrect for validation")
				}
			},
		},
		{
			name:           "GetConsentFactory",
			factoryFunc:    GetConsentFactory,
			expectedName:   "getConsent",
			expectedMethod: "GET",
			validate: func(t *testing.T, tc *data_factory.TestCase) {
				if !strings.Contains(tc.Path, member.PersonAccountId) {
					t.Error("Path missing account id")
				}
			},
		},
		{
			name:           "SetConsentFactory",
			factoryFunc:    SetConsentFactory,
			expectedName:   "setConsent",
			expectedMethod: "POST",
			validate: func(t *testing.T, tc *data_factory.TestCase) {
				if !strings.Contains(tc.Body, member.PersonAccountId) {
					t.Error("Body missing account id")
				}
				// Check for random OptIn/OptOut
				if !strings.Contains(tc.Body, "OptIn") && !strings.Contains(tc.Body, "OptOut") {
					t.Error("Body missing random consent status")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the outer factory function
			innerFactory := tt.factoryFunc(instanceURL)

			// Call the returned inner function
			tc := innerFactory(member)

			// Assert basic fields
			if tc.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, tc.Name)
			}
			if tc.Method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, tc.Method)
			}
			if !strings.HasPrefix(tc.Path, instanceURL) {
				t.Errorf("Path does not start with InstanceURL: %s", tc.Path)
			}

			// Run specific validation logic
			tt.validate(t, tc)
		})
	}
}
