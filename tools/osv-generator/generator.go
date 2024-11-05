package osv_generator

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"
)

const URL = "https://security.access.redhat.com/data/csaf/v2/advisories"

// Get list of new advisories from the last `days`
func getAdvisoryList(days int) ([]string, error) {
	response, err := http.Get(fmt.Sprintf("%s/%s", URL, "changes.csv"))
	if err != nil {
		fmt.Println("Error downloading file:", err)
		return nil, err
	}
	defer response.Body.Close()

	// Load all advisories from the document
	csvReader := csv.NewReader(response.Body)
	records, err := csvReader.ReadAll()
	if err != nil {
		fmt.Println("Error parsing CSV:", err)
		return nil, err
	}

	var advisories []string
	dateThreshold := time.Now().AddDate(0, 0, -days)

	// Filter out those which are not older than the specified amount of `days`
	for _, record := range records {
		advisoryDate, err := time.Parse(time.RFC3339, record[1])
		if err != nil {
			fmt.Println("Error parsing date:", err)
			return nil, err
		}

		advisories = append(advisories, record[0])

		if advisoryDate.Before(dateThreshold) {
			fmt.Printf("Found %d new advisories\n", len(advisories))
			break
		}
	}

	return advisories, nil

}

// Extract advisory data from the given URL, store as list of OSV objects
func extractAdvisory(advisory string) []OSV {
	vexVulnerability, err := GetVEXFromUrl(fmt.Sprintf("%s/%s", URL, advisory))
	if err != nil {
		panic(err)
	}
	convertedVulnerabilities := ConvertToOSV(vexVulnerability)
	return convertedVulnerabilities
}

// Generate OSV vulnerabilities from CSAF VEX data and store to a file
func GenerateOSV(filename string) error {
	var osvList []OSV
	advisories, err := getAdvisoryList(1) // Advisories created in the last 24 hours
	if err != nil {
		return err
	}

	// Extract vulnerability data and convert to OSV format
	for _, advisory := range advisories {
		advisoryChan := make(chan []OSV)

		go func(advisory string) {
			advisoryChan <- extractAdvisory(advisory)
		}(advisory)

		osvList = append(osvList, <-advisoryChan...)
	}

	if err := StoreToFile(filename, osvList); err != nil {
		return err
	}
	return nil
}
