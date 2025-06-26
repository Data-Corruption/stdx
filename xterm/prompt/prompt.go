// Package prompt provides functions to interactively prompt the user for input in a terminal.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Int prompts the user for an integer input.
// It re-prompts until a valid integer is entered.
// Returns 0 and prints an error message if reading fails (e.g., EOF).
func Int(prompt string) int {
	reader := bufio.NewReader(os.Stdin)
	fullPrompt := fmt.Sprintf("%s: ", prompt) // Add consistent suffix

	for {
		fmt.Print(fullPrompt)
		input, err := readLine(reader)
		if err != nil {
			// Handle read errors, print message and return 0
			// Don't print error for EOF, just a newline if possible
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "\nError reading input: %v\n", err)
			} else {
				fmt.Println() // Ensure newline after EOF in terminal
			}
			return 0
		}

		// Attempt to parse the integer
		val, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid integer.")
			continue // Re-prompt
		}
		return val // Success
	}
}

// Uint prompts the user for a non-negative integer input.
// It re-prompts until a valid unsigned integer is entered.
// Returns 0 and prints an error message if reading fails (e.g., EOF).
func Uint(prompt string) uint {
	reader := bufio.NewReader(os.Stdin)
	fullPrompt := fmt.Sprintf("%s: ", prompt) // Add consistent suffix

	for {
		fmt.Print(fullPrompt)
		input, err := readLine(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "\nError reading input: %v\n", err)
			} else {
				fmt.Println()
			}
			return 0
		}

		// Attempt to parse the unsigned integer (base 10, default bit size)
		// Use ParseInt and check range to handle negative sign explicitly if needed,
		// or ParseUint directly which handles the negative sign as an error.
		val, err := strconv.ParseUint(input, 10, 0) // Use base 10, bit size 0 for native uint
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid non-negative integer.")
			continue // Re-prompt
		}
		return uint(val) // Success (convert from uint64 if necessary)
	}
}

// String prompts the user for a string input.
// Returns the trimmed string. Returns empty string on read error.
func String(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", prompt) // Add consistent suffix

	input, err := readLine(reader)
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "\nError reading input: %v\n", err)
		return "", err // Return error if not EOF
	}
	// Also return empty string on EOF if nothing was read before it
	if err == io.EOF && input == "" {
		fmt.Println()
		return "", nil
	}
	return input, nil
}

// YesNo asks a yes/no question to the user.
// It accepts "y", "yes", "n", "no" (case-insensitive).
// Re-prompts on invalid input.
// Returns false on read error (e.g., EOF) or if the user enters "n" or "no".
func YesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fullPrompt := fmt.Sprintf("%s (y/n): ", prompt)

	for {
		fmt.Print(fullPrompt)
		input, err := readLine(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "\nError reading input: %v\n", err)
			} else {
				fmt.Println()
			}
			return false // Default to 'no' on read error
		}

		inputLower := strings.ToLower(input)

		switch inputLower {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Invalid input. Please enter 'y', 'yes', 'n', or 'no'.")
			// Loop again to re-prompt
		}
	}
}

// Helper function to read a line from stdin
func readLine(reader *bufio.Reader) (string, error) {
	str, err := reader.ReadString('\n')
	return strings.TrimSpace(str), err
}
