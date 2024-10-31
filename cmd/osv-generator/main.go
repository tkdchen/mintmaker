package main

import (
	"flag"
	"log"

	osv_generator "github.com/konflux-ci/mintmaker/tools/osv-generator"
)

// A demo which parses RPM CVE data into OSV database format based on input CSAF VEX url
// TODO: implement the ability to process all updated advisories
func main() {
	url := flag.String("url", "", "Url pointing to CSAF VEX file")
	filename := flag.String("file", "demo.nedb", "Name of the file to store OSV data")

	flag.Parse()

	if err := osv_generator.GenerateOSV(*url, *filename); err != nil {
		log.Fatalf("Error generating OSV: %v\n", err)
	}
}
