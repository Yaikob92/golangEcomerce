package utils

import (
	"strings"

	"github.com/ua-parser/uap-go/uaparser"
)

type DeviceInfo struct {
	DeviceName      string
	Browser         string
	OperatingSystem string
}

var parser *uaparser.Parser

func init() {
	// Initialize the built-in parser
	parser = uaparser.NewFromSaved()
}

// ParseUserAgent extracts device, browser, and OS info from User-Agent string.
func ParseUserAgent(uaString string) DeviceInfo {
	if uaString == "" {
		return DeviceInfo{
			DeviceName:      "Unknown Device",
			Browser:         "Unknown Browser",
			OperatingSystem: "Unknown OS",
		}
	}

	client := parser.Parse(uaString)

	// Format browser
	browser := client.UserAgent.Family
	if client.UserAgent.Major != "" {
		browser += " " + client.UserAgent.Major
	}

	// Format OS
	os := client.Os.Family
	if client.Os.Major != "" {
		os += " " + client.Os.Major
	}

	// Format device
	device := client.Device.Family
	if device == "Other" || device == "" {
		if strings.Contains(strings.ToLower(uaString), "windows") {
			device = "Windows PC"
		} else if strings.Contains(strings.ToLower(uaString), "macintosh") {
			device = "Mac"
		} else if strings.Contains(strings.ToLower(uaString), "linux") {
			device = "Linux PC"
		} else {
			device = "Unknown Device"
		}
	}

	return DeviceInfo{
		DeviceName:      device,
		Browser:         browser,
		OperatingSystem: os,
	}
}
