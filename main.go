package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func main() {
	// Define command line flags
	urlFlag := flag.String("url", "", "URL to monitor (required)")
	intervalFlag := flag.Int("interval", 60, "Check interval in seconds")
	elfPathFlag := flag.String("elf", "", "Path to ELF binary to execute when website is down (required)")
	timeoutFlag := flag.Int("timeout", 10, "HTTP request timeout in seconds")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	retriesFlag := flag.Int("retries", 3, "Number of retries before considering site down")
	
	flag.Parse()
	
	// Validate required flags
	if *urlFlag == "" {
		log.Fatal("Error: URL is required. Use -url flag.")
	}
	
	if *elfPathFlag == "" {
		log.Fatal("Error: ELF binary path is required. Use -elf flag.")
	}
	
	// Validate that the ELF file exists and is executable
	elfInfo, err := os.Stat(*elfPathFlag)
	if err != nil {
		log.Fatalf("Error: Cannot access ELF binary %s: %v", *elfPathFlag, err)
	}
	
	// Check if file is executable
	if elfInfo.Mode().Perm()&0111 == 0 {
		log.Fatalf("Error: ELF binary %s is not executable", *elfPathFlag)
	}
	
	log.Printf("Starting website monitor for %s", *urlFlag)
	log.Printf("Will execute %s when website is down", *elfPathFlag)
	log.Printf("Checking every %d seconds", *intervalFlag)
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(*timeoutFlag) * time.Second,
	}
	
	// Main monitoring loop
	for {
		siteDown := checkWebsiteDown(*urlFlag, client, *retriesFlag, *verboseFlag)
		
		if siteDown {
			log.Printf("Website %s is DOWN! Executing ELF binary...", *urlFlag)
			executeELF(*elfPathFlag)
		} else if *verboseFlag {
			log.Printf("Website %s is UP", *urlFlag)
		}
		
		// Wait for the specified interval
		time.Sleep(time.Duration(*intervalFlag) * time.Second)
	}
}

// checkWebsiteDown checks if a website is down by making HTTP requests
// Returns true if the website is considered down
func checkWebsiteDown(url string, client *http.Client, retries int, verbose bool) bool {
	for i := 0; i < retries; i++ {
		resp, err := client.Get(url)
		
		if err != nil {
			if verbose {
				log.Printf("Request failed (attempt %d/%d): %v", i+1, retries, err)
			}
			// If not our last attempt, try again
			if i < retries-1 {
				time.Sleep(2 * time.Second) // Small delay between retries
				continue
			}
			return true // Website is down after all retries failed
		}
		
		defer resp.Body.Close()
		
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			if verbose {
				log.Printf("Bad status code (attempt %d/%d): %d", i+1, retries, resp.StatusCode)
			}
			// If not our last attempt, try again
			if i < retries-1 {
				time.Sleep(2 * time.Second) // Small delay between retries
				continue
			}
			return true // Website is down after all retries returned bad status codes
		}
		
		// If we get here, the website is up
		return false
	}
	
	return true // Should not reach here, but if we do, assume the site is down
}

// executeELF runs the specified ELF binary
func executeELF(elfPath string) {
	cmd := exec.Command(elfPath)
	
	// Capture output
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		log.Printf("Failed to execute ELF binary: %v", err)
		return
	}
	
	// Log the output
	fmt.Println("ELF binary output:")
	fmt.Println(string(output))
}
