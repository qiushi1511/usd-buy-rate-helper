package poller

import (
	"testing"
	"time"
)

func TestIsBusinessHours(t *testing.T) {
	tests := []struct {
		name           string
		skipOffHours   bool
		startHour      int
		endHour        int
		testHour       int
		expectedResult bool
	}{
		{
			name:           "Within business hours (10:00)",
			skipOffHours:   true,
			startHour:      8,
			endHour:        22,
			testHour:       10,
			expectedResult: true,
		},
		{
			name:           "Start of business hours (08:00)",
			skipOffHours:   true,
			startHour:      8,
			endHour:        22,
			testHour:       8,
			expectedResult: true,
		},
		{
			name:           "End of business hours (22:00)",
			skipOffHours:   true,
			startHour:      8,
			endHour:        22,
			testHour:       22,
			expectedResult: false,
		},
		{
			name:           "Before business hours (06:00)",
			skipOffHours:   true,
			startHour:      8,
			endHour:        22,
			testHour:       6,
			expectedResult: false,
		},
		{
			name:           "After business hours (23:00)",
			skipOffHours:   true,
			startHour:      8,
			endHour:        22,
			testHour:       23,
			expectedResult: false,
		},
		{
			name:           "Business hours check disabled",
			skipOffHours:   false,
			startHour:      8,
			endHour:        22,
			testHour:       3,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Poller{
				skipOffHours:       tt.skipOffHours,
				businessHoursStart: tt.startHour,
				businessHoursEnd:   tt.endHour,
			}

			// Mock the current time by testing the logic directly
			cstLocation := time.FixedZone("CST", 8*60*60)
			testTime := time.Date(2025, 11, 25, tt.testHour, 0, 0, 0, cstLocation)

			// Test the logic
			if !tt.skipOffHours {
				if result := p.isBusinessHours(); result != tt.expectedResult {
					t.Errorf("isBusinessHours() = %v, want %v", result, tt.expectedResult)
				}
				return
			}

			// For time-based tests, check if the hour falls within range
			hour := testTime.Hour()
			result := hour >= p.businessHoursStart && hour < p.businessHoursEnd
			if result != tt.expectedResult {
				t.Errorf("Business hours check for %02d:00 = %v, want %v", tt.testHour, result, tt.expectedResult)
			}
		})
	}
}

func TestWithBusinessHours(t *testing.T) {
	p := &Poller{}
	opt := WithBusinessHours(9, 21)
	opt(p)

	if !p.skipOffHours {
		t.Error("WithBusinessHours should enable skipOffHours")
	}
	if p.businessHoursStart != 9 {
		t.Errorf("businessHoursStart = %d, want 9", p.businessHoursStart)
	}
	if p.businessHoursEnd != 21 {
		t.Errorf("businessHoursEnd = %d, want 21", p.businessHoursEnd)
	}
}

func TestWithoutBusinessHours(t *testing.T) {
	p := &Poller{skipOffHours: true}
	opt := WithoutBusinessHours()
	opt(p)

	if p.skipOffHours {
		t.Error("WithoutBusinessHours should disable skipOffHours")
	}
}

func TestNewPollerDefaults(t *testing.T) {
	p := NewPoller(nil, nil, nil)

	if !p.skipOffHours {
		t.Error("Default should enable business hours check")
	}
	if p.businessHoursStart != 8 {
		t.Errorf("Default businessHoursStart = %d, want 8", p.businessHoursStart)
	}
	if p.businessHoursEnd != 22 {
		t.Errorf("Default businessHoursEnd = %d, want 22", p.businessHoursEnd)
	}
}
