package properties

import "strings"

func inferRenovationScope(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "kunto"), strings.Contains(value, "survey"), strings.Contains(value, "inspection"), strings.Contains(value, "tutkimus"), strings.Contains(value, "kuvaus"):
		return "survey"
	case strings.Contains(value, "tarveselvitys"), strings.Contains(value, "hankesuunnittelu"), strings.Contains(value, "suunnittelu"), strings.Contains(value, "kilpailutus"):
		return "planning"
	case strings.Contains(value, "sukitus"), strings.Contains(value, "partial"), strings.Contains(value, "ositt"), strings.Contains(value, "huolto"), strings.Contains(value, "maalaus"), strings.Contains(value, "lakkaus"):
		return "partial"
	case strings.Contains(value, "uusittu"), strings.Contains(value, "uusinta"), strings.Contains(value, "saneeraus"), strings.Contains(value, "peruskorjaus"), strings.Contains(value, "full"), strings.Contains(value, "renewal"):
		return "full"
	default:
		return "unknown"
	}
}

func inferRenovationStage(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "kunnossapitotarveselvitys"), strings.Contains(value, "tarveselvitys"):
		return "need_assessment"
	case strings.Contains(value, "kuntotutkimus"), strings.Contains(value, "kartoitus"), strings.Contains(value, "kuvaus"):
		return "condition_survey"
	case strings.Contains(value, "hankesuunnittelu"), strings.Contains(value, "suunnittelu"):
		return "project_planning"
	case strings.Contains(value, "kilpailutus"):
		return "tendering"
	case strings.Contains(value, "päätös"), strings.Contains(value, "paatos"):
		return "decision"
	case strings.Contains(value, "urakka"), strings.Contains(value, "toteutus"):
		return "execution"
	case strings.Contains(value, "huolto"):
		return "maintenance"
	default:
		return "unknown"
	}
}

func inferRenovationResponsibility(value string) string {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "osakas"), strings.Contains(value, "osakkaan vastuulla"):
		return "shareholder"
	case strings.Contains(value, "taloyhtiö"), strings.Contains(value, "taloyhtio"), strings.Contains(value, "kiinteistö"), strings.Contains(value, "kiinteisto"):
		return "housing_company"
	default:
		return "unknown"
	}
}
