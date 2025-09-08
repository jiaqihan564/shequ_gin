package utils

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"invalid-email", false},
		{"@example.com", false},
		{"test@", false},
		{"", false},
		{"test@.com", false},
	}

	for _, test := range tests {
		result := ValidateEmail(test.email)
		if result != test.expected {
			t.Errorf("ValidateEmail(%s) = %v, expected %v", test.email, result, test.expected)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		expected bool
	}{
		{"validuser", true},
		{"user123", true},
		{"user_name", true},
		{"ab", false},                    // 太短
		{"thisusernameistoolong", false}, // 太长
		{"123user", false},               // 以数字开头
		{"user-name", false},             // 包含连字符
		{"", false},
	}

	for _, test := range tests {
		result := ValidateUsername(test.username)
		if result != test.expected {
			t.Errorf("ValidateUsername(%s) = %v, expected %v", test.username, result, test.expected)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		expected bool
	}{
		{"password123", true},
		{"Pass1234", true},
		{"short", false},    // 太短
		{"password", false}, // 没有数字
		{"12345678", false}, // 没有字母
		{"", false},
	}

	for _, test := range tests {
		result := ValidatePassword(test.password)
		if result != test.expected {
			t.Errorf("ValidatePassword(%s) = %v, expected %v", test.password, result, test.expected)
		}
	}
}

func TestValidatePasswordStrong(t *testing.T) {
	tests := []struct {
		password string
		expected bool
	}{
		{"StrongPass123!", true},
		{"MySecure@Pass1", true},
		{"weakpass", false},     // 太短且不符合要求
		{"Password123", false},  // 没有特殊字符
		{"password123!", false}, // 没有大写字母
		{"PASSWORD123!", false}, // 没有小写字母
		{"Password!", false},    // 没有数字
	}

	for _, test := range tests {
		result := ValidatePasswordStrong(test.password)
		if result != test.expected {
			t.Errorf("ValidatePasswordStrong(%s) = %v, expected %v", test.password, result, test.expected)
		}
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		phone    string
		expected bool
	}{
		{"13812345678", true},
		{"15987654321", true},
		{"12345678901", false},  // 不是以1开头
		{"1381234567", false},   // 太短
		{"138123456789", false}, // 太长
		{"", false},
	}

	for _, test := range tests {
		result := ValidatePhone(test.phone)
		if result != test.expected {
			t.Errorf("ValidatePhone(%s) = %v, expected %v", test.phone, result, test.expected)
		}
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello\tworld", "hello\tworld"},
		{"hello\x00world", "helloworld"}, // 控制字符被移除
		{"", ""},
	}

	for _, test := range tests {
		result := SanitizeString(test.input)
		if result != test.expected {
			t.Errorf("SanitizeString(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
