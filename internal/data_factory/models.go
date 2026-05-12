// Package data_factory handles data preparation and randomization
package data_factory

// MemberDetails in use as the main data model of the testing logic
type MemberDetails struct {
	ID               string `json:"Id"`
	MembershipNumber string `json:"MembershipNumber"`
	PersonAccountID  string `json:"PersonAccount__c"`
	Vouchers         []Voucher
}

// Voucher is a supportive struct for MemberDetails
type Voucher struct {
	VoucherCode string
}

// TestCase is the struct stores the test details to be executed
type TestCase struct {
	Name   string
	Method string
	Path   string
	Body   string
}

// ActivateVoucherRequest is the struct for the API request
type ActivateVoucherRequest struct {
	MembershipNumber      string `json:"membershipNumber"`
	VoucherDefinitionID   string `json:"VoucherDefinitionId"`
	VoucherDefinitionCode string `json:"VoucherDefinitionCode"`
	VoucherID             string `json:"VoucherId"`
	IsActive              bool   `json:"isActive"`
	Language              string `json:"language"`
}

// AuthResponse is the struct storing the response from Salesforce oauth
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
}
