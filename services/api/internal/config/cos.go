package config

import (
	"net/url"
	"strings"
)

// COSMaterial contains public bucket coordinates only. COS credentials come
// from the CVM role and never enter environment-backed runtime configuration.
type COSMaterial struct {
	bucket       string
	region       string
	publicOrigin string
}

func (material COSMaterial) Bucket() string       { return material.bucket }
func (material COSMaterial) Region() string       { return material.region }
func (material COSMaterial) PublicOrigin() string { return material.publicOrigin }

// ParseCOSMaterial validates the exact production bucket and public-read
// origin used by the object-store adapter.
func ParseCOSMaterial(bucket, region, publicOrigin string) (COSMaterial, error) {
	if !validCOSBucket(bucket) || !validTencentRegion(region) {
		return COSMaterial{}, loadError{reason: "production_cos_configuration_invalid"}
	}
	origin, err := url.ParseRequestURI(publicOrigin)
	if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || origin.Port() != "" ||
		origin.User != nil || (origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		return COSMaterial{}, loadError{reason: "production_cos_configuration_invalid"}
	}
	return COSMaterial{bucket: bucket, region: region, publicOrigin: strings.TrimRight(publicOrigin, "/")}, nil
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
