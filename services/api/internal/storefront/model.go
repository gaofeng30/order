package storefront

import (
	"math"
	"strings"
	"unicode/utf8"
)

type BusinessStatus string

const (
	BusinessOpen   BusinessStatus = "open"
	BusinessClosed BusinessStatus = "closed"
	BusinessCutoff BusinessStatus = "cutoff"
)

type Settings struct {
	StoreName      string
	StoreAddress   string
	PickupPoint    string
	Announcement   string
	BusinessStatus BusinessStatus
	LaunchLayer    *LaunchLayer
	Flavors        []string
}

// LaunchLayer contains durable object-store facts only. Public URLs are
// resolved at the HTTP read boundary and are never persisted in MySQL.
type LaunchLayer struct {
	ImageObjectKey string
	CenterX        float64
	CenterY        float64
	WidthRatio     float64
	AspectRatio    float64
}

func (settings Settings) valid() bool {
	if !validRequiredText(settings.StoreName) || !validRequiredText(settings.StoreAddress) || !validRequiredText(settings.PickupPoint) {
		return false
	}
	if !utf8.ValidString(settings.Announcement) || utf8.RuneCountInString(settings.Announcement) > 1000 || settings.Flavors == nil {
		return false
	}
	switch settings.BusinessStatus {
	case BusinessOpen, BusinessClosed, BusinessCutoff:
	default:
		return false
	}
	if settings.LaunchLayer != nil && !settings.LaunchLayer.valid() {
		return false
	}
	seen := make(map[string]struct{}, len(settings.Flavors))
	for _, flavor := range settings.Flavors {
		if !validRequiredText(flavor) {
			return false
		}
		if _, duplicate := seen[flavor]; duplicate {
			return false
		}
		seen[flavor] = struct{}{}
	}
	return true
}

func validRequiredText(value string) bool {
	return utf8.ValidString(value) && value != "" && strings.TrimSpace(value) == value
}

func (layer LaunchLayer) valid() bool {
	values := []float64{layer.CenterX, layer.CenterY, layer.WidthRatio, layer.AspectRatio}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return validObjectKey(layer.ImageObjectKey) &&
		layer.CenterX >= 0 && layer.CenterX <= 1 &&
		layer.CenterY >= 0 && layer.CenterY <= 1 &&
		layer.WidthRatio > 0 && layer.WidthRatio <= 1 &&
		layer.AspectRatio > 0
}

func validObjectKey(value string) bool {
	return utf8.ValidString(value) && len(value) >= 1 && len(value) <= 1024
}
