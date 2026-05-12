// Package throughput contains the test case factories for throughput tests and Execution logic. Each factory function returns a function that takes in MemberDetails and returns a TestCase struct with the appropriate API endpoint, method, and body for the test case.
package throughput

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/aliozdemir13/clockwork-piranha/internal/data_factory"
)

// GetVoucherFactory returns a function that generates a TestCase for the getVouchers API endpoint using the provided InstanceURL and MemberDetails.
func GetVoucherFactory(InstanceURL string) func(*data_factory.MemberDetails) *data_factory.TestCase {
	return func(m *data_factory.MemberDetails) *data_factory.TestCase {

		return &data_factory.TestCase{
			Name:   "getVouchers",
			Method: "GET",
			Path:   fmt.Sprintf("%s/services/apexrest/LoyaltyVouchers/v1?membershipNumber=%s&language=de", InstanceURL, m.MembershipNumber),
			Body:   "",
		}
	}
}

// ActivateVoucherFactory returns a function that generates a TestCase for the activateVoucher API endpoint using the provided InstanceURL and MemberDetails.
func ActivateVoucherFactory(InstanceURL string) func(*data_factory.MemberDetails) *data_factory.TestCase {
	return func(m *data_factory.MemberDetails) *data_factory.TestCase {
		v := m.Vouchers[rand.IntN(len(m.Vouchers))]

		req := data_factory.ActivateVoucherRequest{
			MembershipNumber:      m.MembershipNumber,
			VoucherDefinitionCode: v.VoucherCode,
			IsActive:              true,
			Language:              "de",
		}
		b, _ := json.Marshal(req)

		return &data_factory.TestCase{
			Name:   "activateVoucher",
			Method: "POST",
			Path:   InstanceURL + "/services/apexrest/LoyaltyVouchers/v1/SetLoyaltyVoucherStatus",
			Body:   string(b),
		}
	}
}

// ValidateVoucherFactory returns a function that generates a TestCase for the validateVoucher API endpoint using the provided InstanceURL and MemberDetails.
func ValidateVoucherFactory(InstanceURL string) func(*data_factory.MemberDetails) *data_factory.TestCase {
	return func(m *data_factory.MemberDetails) *data_factory.TestCase {

		req := data_factory.MemberDetails{
			MembershipNumber: m.MembershipNumber,
			Vouchers:         []data_factory.Voucher{m.Vouchers[rand.IntN(len(m.Vouchers))]},
		}
		b, _ := json.Marshal(req)

		return &data_factory.TestCase{
			Name:   "validateVouchers",
			Method: "POST",
			Path:   InstanceURL + "/services/apexrest/LoyaltyVouchers/v1/ValidateVoucherRedemption",
			Body:   string(b),
		}
	}
}

// GetConsentFactory returns a function that generates a TestCase for the getConsent API endpoint using the provided InstanceURL and MemberDetails.
func GetConsentFactory(InstanceURL string) func(*data_factory.MemberDetails) *data_factory.TestCase {
	return func(m *data_factory.MemberDetails) *data_factory.TestCase {

		return &data_factory.TestCase{
			Name:   "getConsent",
			Method: "GET",
			Path:   InstanceURL + "/services/apexrest/Consent/v1?AccountId=" + m.PersonAccountID,
			Body:   "",
		}
	}
}

// SetConsentFactory returns a function that generates a TestCase for the setConsent API endpoint using the provided InstanceURL and MemberDetails.
func SetConsentFactory(InstanceURL string) func(*data_factory.MemberDetails) *data_factory.TestCase {
	return func(m *data_factory.MemberDetails) *data_factory.TestCase {
		var status string

		arr := []string{
			"OptIn",
			"OptOut",
		}

		status = arr[rand.IntN(len(arr))]

		return &data_factory.TestCase{
			Name:   "setConsent",
			Method: "POST",
			Path:   InstanceURL + "/services/apexrest/Consent/v1",
			Body:   fmt.Sprintf(`{"AccountId" : "%s","customerConsents" :[{"Consent" : "%s","DataUsePurpose" : "Newsletter","Channel": "Email","Source" : "App"}]}`, m.PersonAccountID, status),
		}
	}
}
