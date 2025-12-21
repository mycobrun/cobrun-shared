// Package fixtures provides test data for unit and integration tests.
package fixtures

import (
	"time"

	"github.com/google/uuid"
)

// UserFixture represents a test user.
type UserFixture struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Phone            string    `json:"phone"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	UserType         string    `json:"user_type"`
	Status           string    `json:"status"`
	EmailVerified    bool      `json:"email_verified"`
	PhoneVerified    bool      `json:"phone_verified"`
	PasswordHash     string    `json:"password_hash"`
	ReferralCode     string    `json:"referral_code"`
	CreatedAt        time.Time `json:"created_at"`
}

// TestUsers provides common user fixtures for testing.
var TestUsers = struct {
	Rider          UserFixture
	Driver         UserFixture
	Admin          UserFixture
	UnverifiedUser UserFixture
	SuspendedUser  UserFixture
}{
	Rider: UserFixture{
		ID:            "test-rider-001",
		Email:         "testrider@cobrun.com",
		Phone:         "+14155551234",
		FirstName:     "Test",
		LastName:      "Rider",
		UserType:      "rider",
		Status:        "active",
		EmailVerified: true,
		PhoneVerified: true,
		PasswordHash:  "$2a$10$XYZ...", // bcrypt hash of "password123"
		ReferralCode:  "RIDER001",
		CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
	},
	Driver: UserFixture{
		ID:            "test-driver-001",
		Email:         "testdriver@cobrun.com",
		Phone:         "+14155555678",
		FirstName:     "Test",
		LastName:      "Driver",
		UserType:      "driver",
		Status:        "active",
		EmailVerified: true,
		PhoneVerified: true,
		PasswordHash:  "$2a$10$XYZ...",
		ReferralCode:  "DRIVER01",
		CreatedAt:     time.Now().Add(-60 * 24 * time.Hour),
	},
	Admin: UserFixture{
		ID:            "test-admin-001",
		Email:         "testadmin@cobrun.com",
		Phone:         "+14155559999",
		FirstName:     "Test",
		LastName:      "Admin",
		UserType:      "admin",
		Status:        "active",
		EmailVerified: true,
		PhoneVerified: true,
		PasswordHash:  "$2a$10$XYZ...",
		ReferralCode:  "ADMIN001",
		CreatedAt:     time.Now().Add(-90 * 24 * time.Hour),
	},
	UnverifiedUser: UserFixture{
		ID:            "test-unverified-001",
		Email:         "unverified@cobrun.com",
		Phone:         "+14155551111",
		FirstName:     "Unverified",
		LastName:      "User",
		UserType:      "rider",
		Status:        "pending",
		EmailVerified: false,
		PhoneVerified: false,
		PasswordHash:  "$2a$10$XYZ...",
		ReferralCode:  "UNVER001",
		CreatedAt:     time.Now().Add(-1 * 24 * time.Hour),
	},
	SuspendedUser: UserFixture{
		ID:            "test-suspended-001",
		Email:         "suspended@cobrun.com",
		Phone:         "+14155552222",
		FirstName:     "Suspended",
		LastName:      "User",
		UserType:      "rider",
		Status:        "suspended",
		EmailVerified: true,
		PhoneVerified: true,
		PasswordHash:  "$2a$10$XYZ...",
		ReferralCode:  "SUSPEN01",
		CreatedAt:     time.Now().Add(-45 * 24 * time.Hour),
	},
}

// NewRandomUser creates a new random user fixture.
func NewRandomUser(userType string) UserFixture {
	id := uuid.New().String()
	return UserFixture{
		ID:            id,
		Email:         "user-" + id[:8] + "@cobrun.com",
		Phone:         "+1415555" + id[:4],
		FirstName:     "User",
		LastName:      id[:8],
		UserType:      userType,
		Status:        "pending",
		EmailVerified: false,
		PhoneVerified: false,
		PasswordHash:  "$2a$10$XYZ...",
		ReferralCode:  "REF" + id[:5],
		CreatedAt:     time.Now(),
	}
}
