// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Initialize the map
	testResults := make(map[string]string)
	// Initialize counters
	totalTests := 0
	statusCount := make(map[string]int)

	// Create a new Scanner
	scanner := bufio.NewScanner(os.Stdin)

	// Iterate over every line in the input
	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains PASS:, FAIL: or SKIP:
		if strings.Contains(line, "PASS:") || strings.Contains(line, "FAIL:") || strings.Contains(line, "SKIP:") {
			words := strings.Fields(line)
			for i, word := range words {
				if word == "PASS:" || word == "FAIL:" || word == "SKIP:" {
					if i+1 < len(words) {
						testName := words[i+1]
						// Check if the test name already exists in the map
						if _, ok := testResults[testName]; ok {
							panic(fmt.Sprintf("Duplicate test name: %s", testName))
						}

						// Store the result in the map
						testResults[testName] = strings.TrimSuffix(word, ":")
						// Increase counters
						totalTests++
						statusCount[strings.TrimSuffix(word, ":")]++
					}
					break
				}
			}
		}
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}

	// Print the test results
	for testName, result := range testResults {
		fmt.Printf("Test %s: %s\n", testName, result)
	}

	// Print the total test counts
	fmt.Printf("Total tests: %d\n", totalTests)
	for status, count := range statusCount {
		fmt.Printf("Total %s tests: %d\n", status, count)
	}
}
