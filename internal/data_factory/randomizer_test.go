package data_factory

import (
	"testing"
)

func TestShuffleOnInit(t *testing.T) {
	tests := []struct {
		name    string
		input   []*MemberDetails
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Empty slice returns error",
			input:   []*MemberDetails{},
			wantErr: true,
			errMsg:  "sample list is empty",
		},
		{
			name:    "Nil slice returns error",
			input:   nil,
			wantErr: true,
			errMsg:  "sample list is empty",
		},
		{
			name: "Single element slice succeeds",
			input: []*MemberDetails{
				{ID: "1"},
			},
			wantErr: false,
		},
		{
			name: "Multiple elements slice succeeds",
			input: []*MemberDetails{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
				{ID: "4"},
				{ID: "5"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// record original length for validation
			originalLen := len(tt.input)

			// track IDs to ensure no data was lost or corrupted during shuffle
			originalIDs := make(map[string]bool)
			for _, m := range tt.input {
				originalIDs[m.ID] = true
			}

			got, err := ShuffleOnInit(tt.input)

			// check Error State
			if (err != nil) != tt.wantErr {
				t.Errorf("ShuffleOnInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("ShuffleOnInit() error message = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}

			// check Result Integrity
			if len(got) != originalLen {
				t.Errorf("ShuffleOnInit() length changed: got %d, want %d", len(got), originalLen)
			}

			// ensure all original IDs are still present
			for _, m := range got {
				if !originalIDs[m.ID] {
					t.Errorf("ShuffleOnInit() result contains unexpected ID: %s", m.ID)
				}
			}
		})
	}
}
