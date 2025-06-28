// Package prompt provides functions for asking interactive questions in the terminal.
//
// Available functions:
//   - [Int]    Re-prompts until the user enters any signed integer.
//   - [Uint]   Re-prompts until the user enters a non-negative integer.
//   - [String] Reads a single line of text (empty string allowed).
//   - [YesNo]  Asks a yes/no question; returns true when the answer is “yes”.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// exported

// Int prompts the user until a valid integer is entered or error occurs.
func Int(p string) (int, error) { return intR(os.Stdin, p) }

// Uint prompts the user until a valid unsigned integer is entered or error occurs.
func Uint(p string) (uint, error) { return uintR(os.Stdin, p) }

// String prompts the user until a string is entered or error occurs.
func String(p string) (string, error) { return stringR(os.Stdin, p) }

// YesNo asks a yes/no question to the user until a (y/n) response is given or an error occurs.
func YesNo(p string) (bool, error) { return yesNoR(os.Stdin, p) }

// internal

func intR(r io.Reader, prompt string) (int, error) {
	reader := bufio.NewReader(r)
	fullPrompt := fmt.Sprintf("%s: ", prompt)
	// loop until valid input is received
	for {
		fmt.Print(fullPrompt)
		input, err := readLine(reader)
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("error reading input: %w", err)
		}
		if err == io.EOF && input == "" {
			fmt.Println("No input provided. Please enter a valid integer.")
			continue
		}
		// attempt to parse input
		val, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid integer.")
			continue
		}
		return val, nil
	}
}

func uintR(r io.Reader, prompt string) (uint, error) {
	reader := bufio.NewReader(r)
	fullPrompt := fmt.Sprintf("%s: ", prompt)
	// loop until valid input is received
	for {
		fmt.Print(fullPrompt)
		input, err := readLine(reader)
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("error reading input: %w", err)
		}
		if err == io.EOF && input == "" {
			fmt.Println("No input provided. Please enter a valid non-negative integer.")
			continue
		}
		// attempt to parse the input
		val, err := strconv.ParseUint(input, 10, 0) // input, base 10, 0 (native uint)
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid non-negative integer.")
			continue // re-prompt
		}
		return uint(val), nil
	}
}

func stringR(r io.Reader, prompt string) (string, error) {
	reader := bufio.NewReader(r)
	fmt.Printf("%s: ", prompt)
	// receive input
	input, err := readLine(reader)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("error reading input: %w", err)
	}
	// handle empty input
	if err == io.EOF && input == "" {
		fmt.Println()
		return "", nil
	}
	return input, nil
}

func yesNoR(r io.Reader, prompt string) (bool, error) {
	reader := bufio.NewReader(r)
	fullPrompt := fmt.Sprintf("%s (y/n): ", prompt)
	// loop until valid input is received
	for {
		fmt.Print(fullPrompt)
		input, err := readLine(reader)
		if err != nil && err != io.EOF {
			return false, fmt.Errorf("error reading input: %w", err)
		}
		// handle input
		switch strings.ToLower(input) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Invalid input. Please enter one of: 'y', 'yes', 'n', or 'no'.")
		}
	}
}

// Helper function to read a line from stdin
func readLine(reader *bufio.Reader) (string, error) {
	str, err := reader.ReadString('\n')
	return strings.TrimSpace(str), err
}
