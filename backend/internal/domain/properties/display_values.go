package properties

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	providerEnergyCodePattern = regexp.MustCompile(`(?i)^E([0-9]{2})_([A-H])$`)
	compactEnergyCodePattern  = regexp.MustCompile(`(?i)^([A-H])((?:19|20|21)[0-9]{2})$`)
	embeddedEnergyCodePattern = regexp.MustCompile(`(?i)(^|[^[:alnum:]])([A-H])\s*_?\s*((?:19|20|21)[0-9]{2})([^[:alnum:]]|$)`)
)

func displayEnergyClass(values ...string) string {
	for _, value := range values {
		trimmed := cleanDisplayString(value)
		if trimmed == "" {
			continue
		}
		if compactEnergyCodePattern.MatchString(trimmed) {
			return strings.ToUpper(trimmed[:1]) + trimmed[1:]
		}
		if match := embeddedEnergyCodePattern.FindStringSubmatch(trimmed); len(match) == 5 {
			return strings.ToUpper(match[2]) + match[3]
		}
		if match := providerEnergyCodePattern.FindStringSubmatch(trimmed); len(match) == 3 {
			year := "20" + match[1]
			if match[1] >= "50" {
				year = "19" + match[1]
			}
			return strings.ToUpper(match[2]) + year
		}
		return displayEnumValue(trimmed)
	}
	return ""
}

func displayCondition(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "good", "hyvä", "hyva":
		return "Good"
	case "satisfactory", "tyyd", "tyydyttävä", "tyydyttava":
		return "Satisfactory"
	case "poor", "bad", "huono", "tolerable", "välttävä", "valttava":
		return "Poor"
	default:
		return displayEnumValue(value)
	}
}

func displayPropertyType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "kt", "kerrostalo", "apartment", "apartment_house", "apartment_block", "balcony_access_block", "block_of_flats", "flat", "wooden_house_apartment":
		return "Apartment house"
	case "rt", "rivitalo", "row_house", "semi_detached_house", "terraced_house", "terrace_house":
		return "Row house"
	case "ok", "omakotitalo", "detached_house", "separate_house", "single_family_house":
		return "Detached house"
	default:
		return displayEnumValue(value)
	}
}

func displayEnumValue(value string) string {
	trimmed := cleanDisplayString(value)
	if trimmed == "" {
		return ""
	}
	words := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	if len(words) == 0 {
		return trimmed
	}
	for i, word := range words {
		lower := strings.ToLower(word)
		runes := []rune(lower)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
