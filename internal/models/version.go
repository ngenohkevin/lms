package models

import "fmt"

// APIVersion represents an API version
type APIVersion struct {
	Major int
	Minor int
	Patch int
}

// String returns the string representation of the API version
func (v APIVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two API versions
// Returns -1 if v < other, 0 if v == other, 1 if v > other
func (v APIVersion) Compare(other APIVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	return 0
}