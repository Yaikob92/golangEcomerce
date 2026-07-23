package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GenerateSlug converts a name to a URL-friendly slug.
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	var result []rune
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	slug = strings.Trim(string(result), "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}

// GenerateShortID returns a short random numeric string.
func GenerateShortID() string {
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

// GenerateProductSKU generates a SKU from a product name.
func GenerateProductSKU(name string) string {
	runes := []rune(strings.ReplaceAll(strings.ToUpper(name), " ", ""))
	if len(runes) > 6 {
		runes = runes[:6]
	}
	return fmt.Sprintf("PROD-%s", string(runes))
}
