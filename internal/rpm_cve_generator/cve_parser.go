package rpm_cve_generator

import (
	"fmt"
	"time"
)

func getReferencesList(vulnerability *Vulnerability) []*Reference {
	var references []*Reference

	for _, reference := range vulnerability.References {
		if reference.Category == "self" {
			references = append(references, &Reference{
				Type: "REPORT",
				Url:  reference.Url,
			})
		} else {
			references = append(references, &Reference{
				Type: "WEB",
				Url:  reference.Url,
			})
		}
	}
	return references
}

func getDetails(vulnerability *Vulnerability) string {
	for _, note := range vulnerability.Notes {
		if note.Category == "description" {
			return note.Text
		}
	}
	panic("No CVE details found")
}

func getSummary(vulnerability *Vulnerability) string {
	for _, note := range vulnerability.Notes {
		if note.Category == "summary" {
			return note.Text
		}
	}
	panic("No CVE summary found")
}

func getPublishedDate(vulnerability *Vulnerability) string {
	t, err := time.Parse("2006-01-02T15:04:05+00:00", vulnerability.DiscoveryDate)
	if err != nil {
		fmt.Printf("Error parsing time: %v\n", err)
		panic(err)
	}
	return t.Format("2006-01-02T15:04:05Z")
}
