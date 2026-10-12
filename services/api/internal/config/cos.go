package config

import (
	"strings"
)

// COSMaterial contains public bucket coordinates only. COS credentials come
// from the CVM role and never enter environment-backed runtime configuration.
type COSMaterial struct {
	bucket string
	region string
}

func (material COSMaterial) Bucket() string { return material.bucket }
func (material COSMaterial) Region() string { return material.region }

// ParseCOSMaterial validates the exact private production bucket coordinates.
// Read and write access is always issued as a short-lived signed URL.
func ParseCOSMaterial(bucket, region string) (COSMaterial, error) {
	if !validCOSBucket(bucket) || !validTencentRegion(region) {
		return COSMaterial{}, loadError{reason: "production_cos_configuration_invalid"}
	}
	return COSMaterial{bucket: bucket, region: region}, nil
}

func validCOSBucket(value string) bool {
	if len(value) < 7 || len(value) > 63 || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	separator := strings.LastIndexByte(value, '-')
	if separator < 1 || separator == len(value)-1 {
		return false
	}
	appID := value[separator+1:]
	if len(appID) < 5 || len(appID) > 20 || appID[0] == '0' {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	for index := 0; index < len(appID); index++ {
		if appID[index] < '0' || appID[index] > '9' {
			return false
		}
	}
	return true
}
