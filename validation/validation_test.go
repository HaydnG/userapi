package validation

import "testing"

// pkg: userapi/validation
// cpu: AMD Ryzen 7 5800X3D 8-Core Processor
// BenchmarkNumber-16    	43626686	        27.65 ns/op	       0 B/op	       0 allocs/op
func BenchmarkNumber(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Number("1234567890", "0987654321", "1122334455")
	}
}

// TestNumber tests the Number function.
func TestNumber(t *testing.T) {
	tests := []struct {
		inputs   []string
		expected bool
	}{
		{[]string{"123"}, true},
		{[]string{"123", "456"}, true},
		{[]string{"123", "456a"}, false},
		{[]string{"", "456"}, false},
		{[]string{" 123 "}, false},
		{[]string{"-123"}, false},
		{[]string{"123.456"}, false},
		{[]string{"0"}, true},
		{[]string{"123", ""}, false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			result := Number(test.inputs...)
			if result != test.expected {
				t.Errorf("Number(%v) = %v; want %v", test.inputs, result, test.expected)
			}
		})
	}
}

// pkg: userapi/validation
// cpu: AMD Ryzen 7 5800X3D 8-Core Processor
// BenchmarkUser-16    	 3112676	       372.2 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		User("John", "Doe", "jdoe", "Password1", "USA", "john.doe@example.com")
	}
}

// TestUser ensures the resiliance of our user validation
func TestUser(t *testing.T) {
	tests := []struct {
		firstName string
		lastName  string
		nickName  string
		password  string
		country   string
		email     string
		expected  error
	}{
		// Valid user
		{"John", "Doe", "jdoe", "Password1", "USA", "john.doe@csgo.com", nil},
		// Valid user
		{"Lina", "Inverse", "LinaDota", "FireMage1", "Russia", "lina.inverse@dota2.com", nil},
		// First name tests
		{"", "Inferno", "smokeMaster", "Inferno123", "Italy", "inferno@csgo.com", ErrInvalidFirstName},
		{"Pudge", "", "meathook", "Rot12345", "Ukraine", "pudge@dota2.com", ErrInvalidLastName},
		// Nickname tests
		{"Kenny", "S", "", "AWPshot1", "France", "kenny.s@csgo.com", ErrInvalidNickname},
		{"Invoker", "Carl", "Invoker123", "QuasWex1", "Egypt", "invoker@dota2.com", nil},
		// Password tests
		{"Zeus", "Elektra", "thunder", "short", "Greece", "zeus@csgo.com", ErrInvalidPassword},
		{"Crystal", "Maiden", "crystal", "noforcefield1", "Russia", "crystal.maiden@dota2.com", ErrInvalidPassword},
		// Email tests
		{"Neo", "Pro", "neopro", "BestPlayer1", "Denmark", "invalid-email", ErrInvalidEmail},
		{"Juggernaut", "Yurnero", "maskedwarrior", "BladeFury1", "Japan", "yurnero.dota", ErrInvalidEmail},
		// Country tests
		{"Sniper", "Billy", "sharpshooter", "Headshot1", "", "sniper@csgo.com", ErrInvalidCountry},
		{"Anti", "Mage", "AntiMagic", "Blink1234", "Nepal", "anti.mage@dota2.com", nil},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			err := User(test.firstName, test.lastName, test.nickName, test.password, test.country, test.email)
			if err != test.expected {
				t.Errorf("TestCase User %q, %q, %q, %q, %q, %q Got Validation Response = %v; want %v",
					test.firstName, test.lastName, test.nickName, test.password, test.country, test.email, err, test.expected)
			}
		})
	}
}
