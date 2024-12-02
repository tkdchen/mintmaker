package osv_generator

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var vexSampleFile = []byte(`{
    "vulnerabilities": [{
        "cve": "CVE-2022-1234",
        "cwe": {
            "id": "CWE-79"
        },
        "discovery_date": "2022-01-01T00:00:00+00:00",
        "notes": [{
            "category": "summary",
            "text": "Test summary"
        }, {
            "category": "description",
            "text": "Test details"
        }],
        "references": [{
            "category": "self",
            "url": "http://example.com"
        }, {
			"category": "web",
            "url": "http://example2.com"
		}]
    }],
    "document": {
        "aggregate_severity": {
            "text": "High"
        }
    },
    "product_tree": {
        "branches": [{
            "branches": [{
                "category": "architecture",
                "branches": [{
                    "product": {
                        "product_identification_helper": {
                            "purl": "pkg:rpm/testpackage@1.0.1?arch=x86_64"
                        }
                    }
                }, {
                    "product": {
                        "product_identification_helper": {
                            "purl": "pkg:go/fakepackage@1.0.0?arch=x86_64"
                        }
                    }
                }]
            }, {
                "category": "irrelevant",
                "branches": [{
                    "product": {
                        "product_identification_helper": {
                            "purl": "pkg:go/fakepackage@9.9.9?arch=x86_64"
                        }
                    }
                }]
            }]
        }]
    }
}`)
var vexSampleObject VEX

// Initialize CSAF VEX object
func init() {
	if err := json.Unmarshal([]byte(vexSampleFile), &vexSampleObject); err != nil {
		panic(err)
	}
}

func TestGetAffectedList(t *testing.T) {
	affectedList := getAffectedList(vexSampleObject)

	if len(affectedList) != 1 {
		t.Fatalf("expected 1 affected package, got %d", len(affectedList))
	}
	if affectedList[0].Package.Name != "testpackage" {
		t.Fatalf("expected testpackage, got %s", affectedList[0].Package.Name)
	}
	if affectedList[0].Package.Purl != "pkg:rpm/testpackage@1.0.1" {
		t.Fatalf("expected pkg:rpm/testpackage@1.0.0, got %s", affectedList[0].Package.Purl)
	}
}

func TestGetReferencesList(t *testing.T) {
	references := getReferencesList(vexSampleObject.Vulnerabilities[0])
	if len(references) != 2 {
		t.Fatalf("expected 2 references, got %d", len(references))
	}

	if references[0].Type != "REPORT" || references[1].Type != "WEB" {
		t.Fatalf("unexpected reference types: %v", references)
	}
}

func TestGetDetails(t *testing.T) {
	details := getDetails(vexSampleObject.Vulnerabilities[0])
	if details != "Test details" {
		t.Fatalf("expected 'Test summary', got %s", details)
	}
}

func TestGetSummary(t *testing.T) {
	summary := getSummary(vexSampleObject.Vulnerabilities[0])
	if summary != "Test summary" {
		t.Fatalf("expected 'Test description', got %s", summary)
	}
}

func TestGetPublishedDate(t *testing.T) {
	publishedDate := getPublishedDate(vexSampleObject.Vulnerabilities[0])
	expectedDate := "2022-01-01T00:00:00Z"
	if publishedDate != expectedDate {
		t.Fatalf("expected %s, got %s", expectedDate, publishedDate)
	}
}

func TestContains(t *testing.T) {
	affectedList := []*Affected{
		{
			Package: &Package{
				Name: "testpackage",
			},
		},
	}

	affectedPackage := Affected{
		Package: &Package{
			Name: "testpackage",
		},
	}

	if !contains(affectedList, affectedPackage) {
		t.Fatalf("expected package to be contained in the list")
	}

	affectedPackage.Package.Name = "anotherpackage"
	if contains(affectedList, affectedPackage) {
		t.Fatalf("expected package not to be contained in the list")
	}
}

func TestConvertToOSV(t *testing.T) {
	result := OSV{
		SchemaVersion: "1.6.0",
		Id:            "CVE-2022-1234",
		DatabaseSpecific: &DatabaseSpecific{
			Severity: "High",
			CWEids:   []string{"CWE-79"},
		},
		Modified:  "Now",
		Published: "2022-01-01T00:00:00Z",
		Summary:   "Test summary",
		Details:   "Test description",
		References: []*Reference{
			{
				Type: "REPORT",
				Url:  "http://example.com",
			},
			{
				Type: "WEB",
				Url:  "http://example2.com",
			},
		},
		Affected: []*Affected{
			{
				Package: &Package{
					Ecosystem: "Red Hat",
					Name:      "testpackage",
					Purl:      "pkg:rpm/testpackage@1.0.1",
				},
				Ranges: []*Range{
					{
						Type: "ECOSYSTEM",
						Events: []*Event{
							{
								Introduced: "0.0.0",
							},
							{
								Fixed: "1.0.1",
							},
						},
					},
				},
			},
		},
	}

	osv := ConvertToOSV(vexSampleObject)
	if cmp.Equal(osv, result) {
		t.Fatalf("expected %+v, got %+v", result, osv)
	}
}
