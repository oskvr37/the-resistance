package store

import "math/rand/v2"

const charset_letters = "AZWRTYDFGHJKLM"
const charset_numbers = "24679"

func randomCapsString(n int) string {
	if n < 2 {
		panic("n must be at least 2")
	}

	b := make([]byte, n)

	// letters
	for i := 0; i < n-3; i++ {
		b[i] = charset_letters[rand.IntN(len(charset_letters))]
	}

	// numbers
	for i := n - 3; i < n; i++ {
		b[i] = charset_numbers[rand.IntN(len(charset_numbers))]
	}

	return string(b)
}
