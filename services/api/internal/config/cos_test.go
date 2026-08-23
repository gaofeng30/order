package config

import "testing"

func TestParseCOSMaterialAcceptsStrictProductionCoordinates(t *testing.T) {
	material, err := ParseCOSMaterial(
		"order-images-1250000000",
		"ap-guangzhou",
		"https://images.example.com/",
	)
	if err != nil {
		t.Fatal(err)
	}
	if material.Bucket() != "order-images-1250000000" || material.Region() != "ap-guangzhou" || material.PublicOrigin() != "https://images.example.com" {
		t.Fatalf("material=%#v", material)
	}
}

func TestParseCOSMaterialRejectsAmbiguousCoordinates(t *testing.T) {
	tests := []struct {
		name, bucket, region, origin string
	}{
		{name: "bucket without app id", bucket: "order-images", region: "ap-guangzhou", origin: "https://images.example.com"},
		{name: "uppercase bucket", bucket: "Order-images-1250000000", region: "ap-guangzhou", origin: "https://images.example.com"},
		{name: "invalid region", bucket: "order-images-1250000000", region: "guangzhou", origin: "https://images.example.com"},
		{name: "insecure origin", bucket: "order-images-1250000000", region: "ap-guangzhou", origin: "http://images.example.com"},
		{name: "origin with path", bucket: "order-images-1250000000", region: "ap-guangzhou", origin: "https://images.example.com/assets"},
		{name: "origin with credentials", bucket: "order-images-1250000000", region: "ap-guangzhou", origin: "https://user:password@images.example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseCOSMaterial(test.bucket, test.region, test.origin)
			if err == nil || Reason(err) != "production_cos_configuration_invalid" {
				t.Fatalf("err=%v reason=%q", err, Reason(err))
			}
		})
	}
}
