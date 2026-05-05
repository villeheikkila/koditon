package valuation

import (
	"fmt"
	"strings"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := cleanDisplayString(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cleanDisplayString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "<nil>" {
		return ""
	}
	return trimmed
}

func ptrBool(value bool) *bool {
	return &value
}

func compactRenovations(values []BuildingRenovation) []BuildingRenovation {
	out := make([]BuildingRenovation, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value.Kind)) + "|" + strings.ToLower(strings.TrimSpace(value.Component)) + "|"
		if value.Done != nil {
			if *value.Done {
				key += "done|"
			} else {
				key += "planned|"
			}
		}
		if value.Year != nil {
			key += fmt.Sprint(*value.Year)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeRenovationCategory(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(value, "putki"), strings.Contains(value, "pipe"), strings.Contains(value, "lvi"):
		return "pipe"
	case strings.Contains(value, "viem"):
		return "sewer"
	case strings.Contains(value, "vesijohto"), strings.Contains(value, "water"):
		return "water_supply"
	case strings.Contains(value, "julkisivu"), strings.Contains(value, "facade"):
		return "facade"
	case strings.Contains(value, "katto"), strings.Contains(value, "roof"):
		return "roof"
	case strings.Contains(value, "sahko"), strings.Contains(value, "sähkö"), strings.Contains(value, "electric"):
		return "electricity"
	case strings.Contains(value, "ikkuna"), strings.Contains(value, "window"):
		return "window"
	case strings.Contains(value, "parve"), strings.Contains(value, "balcony"):
		return "balcony"
	case strings.Contains(value, "hissi"), strings.Contains(value, "elevator"):
		return "elevator"
	case strings.Contains(value, "lampo"), strings.Contains(value, "lämpö"), strings.Contains(value, "heating"):
		return "heating"
	case strings.Contains(value, "ilmanvaihto"), strings.Contains(value, "ventilation"):
		return "ventilation"
	case strings.Contains(value, "salaoja"), strings.Contains(value, "drainage"):
		return "drainage"
	case strings.Contains(value, "piha"), strings.Contains(value, "yard"):
		return "yard"
	case strings.Contains(value, "yleis"), strings.Contains(value, "common"):
		return "common_area"
	default:
		return value
	}
}

func normalizeRenovationStage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "need_assessment", "condition_survey", "project_planning", "tendering", "decision", "execution", "maintenance", "completed":
		return value
	default:
		return ""
	}
}
