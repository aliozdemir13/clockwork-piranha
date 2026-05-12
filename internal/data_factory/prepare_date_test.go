package data_factory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMembersFromCSV_TableDriven(t *testing.T) {
	// Setup a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		fileContent   string
		setupFileName string // If empty, generated temp file
		wantCount     int
		wantErr       bool
	}{
		{
			name:          "File does not exist",
			setupFileName: "missing.csv",
			wantErr:       true,
		},
		{
			name: "Valid CSV with 2 members",
			fileContent: "id,membership_num,account_id\n" +
				"001,MEM1,ACC1\n" +
				"002,MEM2,ACC2",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "CSV with quotes and spaces",
			fileContent: "id,membership_num,account_id\n" +
				"\"003\",\"MEM 3\",\"ACC 3\"",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "Skip lines with missing columns",
			fileContent: "id,membership_num,account_id\n" +
				"001,MEM1,ACC1\n" + // Valid
				"002,MEM2\n" + // Invalid (only 2 cols)
				"003", // Invalid (only 1 col)
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:        "Empty file (only header)",
			fileContent: "id,membership_num,account_id",
			wantCount:   0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string

			// Logic to handle file creation/missing file
			if tt.setupFileName != "" {
				filePath = filepath.Join(tmpDir, tt.setupFileName)
			} else {
				// Create a physical file with the test content
				tmpFile := filepath.Join(tmpDir, "test.csv")
				err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				filePath = tmpFile
			}

			// Execute the function
			members, err := LoadMembersFromCSV(filePath)

			// Check if we expected an error
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadMembersFromCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check member count
			if len(members) != tt.wantCount {
				t.Errorf("Expected %d members, got %d", tt.wantCount, len(members))
			}

			// Detailed check for the first member if one exists
			if len(members) > 0 {
				if members[0].Vouchers == nil || len(members[0].Vouchers) != 3 {
					t.Errorf("Vouchers not initialized correctly. Got: %v", members[0].Vouchers)
				}
			}
		})
	}
}
