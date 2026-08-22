package storefront

import (
	"math"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"
)

// BusinessStatus is the public storefront operating state.
type BusinessStatus string

const (
	BusinessOpen   BusinessStatus = "open"
	BusinessClosed BusinessStatus = "closed"
	BusinessCutoff BusinessStatus = "cutoff"
)

// Settings is the public singleton storefront configuration.
type Settings struct {
	StoreName      string
	StoreAddress   string
	PickupPoint    string
	Announcement   string
	BusinessStatus BusinessStatus
	LaunchLayer    *LaunchLayer
}

// LaunchLayer is one complete positioned PNG configuration.
type LaunchLayer struct {
	PNGURL      string
	CenterX     float64
	CenterY     float64
	WidthRatio  float64
	AspectRatio float64
}

func (settings Settings) valid() bool {
	if !validRequiredText(settings.StoreName) || !validRequiredText(settings.StoreAddress) || !validRequiredText(settings.PickupPoint) {
		return false
	}
	if !utf8.ValidString(settings.Announcement) || utf8.RuneCountInString(settings.Announcement) > 1000 {
		return false
	}
	switch settings.BusinessStatus {
	case BusinessOpen, BusinessClosed, BusinessCutoff:
		return settings.LaunchLayer == nil || settings.LaunchLayer.valid()
	default:
		return false
	}
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
	return validLaunchPNGURL(layer.PNGURL) &&
		layer.CenterX >= 0 && layer.CenterX <= 1 &&
		layer.CenterY >= 0 && layer.CenterY <= 1 &&
		layer.WidthRatio > 0 && layer.WidthRatio <= 1 &&
		layer.AspectRatio > 0
}

func validLaunchPNGURL(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.ParseUint(port, 10, 16)
		if err != nil || value == 0 {
			return false
		}
	}
	return strings.EqualFold(path.Ext(parsed.Path), ".png")
}
