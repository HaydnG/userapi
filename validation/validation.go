package validation

import (
	"errors"
	"regexp"
	"unicode"
)

// Validation errors
var (
	ErrInvalidFirstName = errors.New("invalid first name")
	ErrInvalidLastName  = errors.New("invalid last name")
	ErrInvalidNickname  = errors.New("invalid nickname")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidCountry   = errors.New("invalid country")
)

// Number checks if the input string is a valid integer without converting it.
func Number(inputs ...string) bool {
	for _, s := range inputs {
		if len(s) == 0 {
			return false
		}

		for _, c := range s {
			if !unicode.IsDigit(c) {
				return false
			}
		}
	}

	return true
}

// User takes in a user object, and enforces validation rules on the user.
func User(firstName, lastName, nickName, password, country, email string) error {
	if !isValidName(firstName) {
		return ErrInvalidFirstName
	}
	if !isValidName(lastName) {
		return ErrInvalidLastName
	}
	if nickName == "" {
		return ErrInvalidNickname
	}
	if !isValidPassword(password) {
		return ErrInvalidPassword
	}
	if !isValidEmail(email) {
		return ErrInvalidEmail
	}
	if country == "" {
		return ErrInvalidCountry
	}
	return nil
}

// isValidName ensures the name is not empty
func isValidName(name string) bool {
	return name != ""
}

// isValidPassword ensures the password meets our policy standards
// Greater than 8
// Contains an upperCase
// Contains a lowerCase
// Contains a number
func isValidPassword(password string) bool {
	var hasMinLen, hasUpper, hasLower, hasNumber bool
	if len(password) >= 8 {
		hasMinLen = true
	}
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		}
	}
	return hasMinLen && hasUpper && hasLower && hasNumber
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidEmail checks if the email meets the standard format
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}
