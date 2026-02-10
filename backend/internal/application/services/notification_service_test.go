package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	db := sqlx.NewDb(mockDB, "postgres")
	return db, mock
}

func TestNewNotificationService(t *testing.T) {
	tests := []struct {
		name             string
		envAccessToken   string
		envPhoneNumberID string
		wantErr          bool
	}{
		{
			name:             "Valid configuration",
			envAccessToken:   "test_token",
			envPhoneNumberID: "123456789",
			wantErr:          false,
		},
		{
			name:             "Missing WhatsApp credentials",
			envAccessToken:   "",
			envPhoneNumberID: "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WHATSAPP_ACCESS_TOKEN", tt.envAccessToken)
			t.Setenv("WHATSAPP_PHONE_NUMBER_ID", tt.envPhoneNumberID)

			db, _ := setupMockDB(t)
			defer db.Close()

			service, err := NewNotificationService(db)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNotificationService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && service == nil {
				t.Error("NewNotificationService() returned nil service")
			}
		})
	}
}

func TestNotificationService_RenderTemplate(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name     string
		template string
		context  *NotificationContext
		want     string
	}{
		{
			name:     "Replace all placeholders",
			template: "Hello {{patient_name}}, your appointment at {{facility_name}} on {{scheduled_date}} at {{scheduled_time}}",
			context: &NotificationContext{
				PatientName:   "John Doe",
				FacilityName:  "Lagos General",
				ScheduledDate: "Monday, Feb 10, 2026",
				ScheduledTime: "2:00 PM",
			},
			want: "Hello John Doe, your appointment at Lagos General on Monday, Feb 10, 2026 at 2:00 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.renderTemplate(tt.template, tt.context)
			if got != tt.want {
				t.Errorf("renderTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotificationService_ExtractTemplateParameters(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name    string
		context *NotificationContext
		want    []string
	}{
		{
			name: "Basic parameters",
			context: &NotificationContext{
				ScheduledDate: "Monday, Feb 10",
				ScheduledTime: "2:00 PM",
				FacilityName:  "Lagos General",
			},
			want: []string{"Monday, Feb 10", "2:00 PM", "Lagos General"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.extractTemplateParameters(tt.context)
			if len(got) != len(tt.want) {
				t.Errorf("extractTemplateParameters() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractTemplateParameters()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
