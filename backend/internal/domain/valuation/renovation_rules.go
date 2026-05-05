package valuation

import (
	"fmt"
	"strings"
)

type renovationForecastRule struct {
	ID                    string
	Category              string
	Label                 string
	HouseTypes            []string
	BuildYearMin          *int32
	BuildYearMax          *int32
	RequiresElevator      *bool
	Lifecycles            []renovationLifecycle
	Severity              string
	ForecastFromBuildYear bool
	WindowYears           int32
	DependsOn             []string
	PriceMechanisms       []string
}

type renovationLifecycle struct {
	Scope           string
	LifespanYears   int32
	Confidence      string
	CreatesFollowUp bool
	FollowUpYears   int32
}

var apartmentHouseRenovationRules = []renovationForecastRule{
	{ID: "apartment.pipe.pre_1975", Category: "pipe", Label: "pipe renovation", HouseTypes: []string{"apartment"}, BuildYearMax: int32Ptr(1975), Lifecycles: renovationLifecycles(50, 25, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"bathroom", "sewer"}, PriceMechanisms: majorRenovationPriceMechanisms()},
	{ID: "apartment.pipe.default", Category: "pipe", Label: "pipe renovation", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(45, 22, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"bathroom", "sewer"}, PriceMechanisms: majorRenovationPriceMechanisms()},
	{ID: "apartment.sewer.default", Category: "sewer", Label: "sewer renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(50, 25, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"pipe", "drainage"}, PriceMechanisms: majorRenovationPriceMechanisms()},
	{ID: "apartment.water_supply.default", Category: "water_supply", Label: "water supply renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(50, 25, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"pipe"}, PriceMechanisms: majorRenovationPriceMechanisms()},
	{ID: "apartment.roof.default", Category: "roof", Label: "roof renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(30, 15, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 4, DependsOn: []string{"facade", "drainage"}, PriceMechanisms: []string{"housing company debt", "maintenance charge pressure", "water damage risk"}},
	{ID: "apartment.facade.default", Category: "facade", Label: "facade renovation", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(30, 15, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"balcony", "window"}, PriceMechanisms: majorRenovationPriceMechanisms()},
	{ID: "apartment.window.default", Category: "window", Label: "window renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(35, 15, 3), Severity: "medium", ForecastFromBuildYear: true, WindowYears: 4, DependsOn: []string{"facade"}, PriceMechanisms: []string{"housing company debt", "energy efficiency", "maintenance charge pressure"}},
	{ID: "apartment.balcony.post_1940", Category: "balcony", Label: "balcony repair", HouseTypes: []string{"apartment"}, BuildYearMin: int32Ptr(1940), Lifecycles: renovationLifecycles(25, 12, 3), Severity: "medium", ForecastFromBuildYear: false, WindowYears: 4, DependsOn: []string{"facade"}, PriceMechanisms: []string{"housing company debt", "use value", "buyer demand"}},
	{ID: "apartment.electricity.default", Category: "electricity", Label: "electrical system renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(40, 20, 3), Severity: "high", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"pipe", "telecom"}, PriceMechanisms: majorRenovationPriceMechanisms()},
	{ID: "apartment.elevator.present", Category: "elevator", Label: "elevator modernization", HouseTypes: []string{"apartment"}, RequiresElevator: ptrBool(true), Lifecycles: renovationLifecycles(30, 15, 3), Severity: "high", ForecastFromBuildYear: false, WindowYears: 4, PriceMechanisms: []string{"housing company debt", "maintenance charge pressure", "accessibility demand"}},
	{ID: "apartment.heating.default", Category: "heating", Label: "heating system renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(25, 12, 3), Severity: "medium", ForecastFromBuildYear: true, WindowYears: 4, DependsOn: []string{"energy"}, PriceMechanisms: []string{"energy efficiency", "maintenance charge pressure", "operational risk"}},
	{ID: "apartment.ventilation.default", Category: "ventilation", Label: "ventilation renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(30, 15, 3), Severity: "medium", ForecastFromBuildYear: true, WindowYears: 4, DependsOn: []string{"energy"}, PriceMechanisms: []string{"indoor air quality", "maintenance charge pressure", "buyer demand"}},
	{ID: "apartment.drainage.default", Category: "drainage", Label: "drainage renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(40, 20, 3), Severity: "medium", ForecastFromBuildYear: true, WindowYears: 5, DependsOn: []string{"yard", "foundation"}, PriceMechanisms: []string{"water damage risk", "housing company debt", "maintenance charge pressure"}},
	{ID: "apartment.yard.default", Category: "yard", Label: "yard renovation", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(20, 10, 3), Severity: "medium", ForecastFromBuildYear: false, WindowYears: 3, PriceMechanisms: []string{"use value", "maintenance charge pressure"}},
	{ID: "apartment.common_area.default", Category: "common_area", Label: "common area renewal", HouseTypes: []string{"apartment"}, Lifecycles: renovationLifecycles(25, 12, 3), Severity: "low", ForecastFromBuildYear: false, WindowYears: 3, PriceMechanisms: []string{"use value", "buyer demand"}},
}

func renovationRulesForBuilding(building BuildingDetails) []renovationForecastRule {
	out := make([]renovationForecastRule, 0, len(apartmentHouseRenovationRules))
	for _, rule := range apartmentHouseRenovationRules {
		if renovationRuleApplies(rule, building) {
			out = append(out, rule)
		}
	}
	return out
}

func renovationRuleForBuilding(category string, building BuildingDetails) (renovationForecastRule, bool) {
	category = normalizeRenovationCategory(category)
	for _, rule := range renovationRulesForBuilding(building) {
		if rule.Category == category {
			return rule, true
		}
	}
	return renovationForecastRule{}, false
}

func renovationRuleApplies(rule renovationForecastRule, building BuildingDetails) bool {
	if len(rule.HouseTypes) > 0 && !renovationHouseTypeMatches(rule.HouseTypes, building) {
		return false
	}
	if rule.BuildYearMin != nil && (building.BuildYear == nil || *building.BuildYear < *rule.BuildYearMin) {
		return false
	}
	if rule.BuildYearMax != nil && (building.BuildYear == nil || *building.BuildYear > *rule.BuildYearMax) {
		return false
	}
	if rule.RequiresElevator != nil && (building.Elevator == nil || *building.Elevator != *rule.RequiresElevator) {
		return false
	}
	return true
}

func renovationHouseTypeMatches(expected []string, building BuildingDetails) bool {
	actual := normalizedRenovationHouseType(building)
	if actual == "" {
		actual = "apartment"
	}
	for _, value := range expected {
		if actual == value {
			return true
		}
	}
	return false
}

func normalizedRenovationHouseType(building BuildingDetails) string {
	value := strings.ToLower(building.BuildingType + " " + building.BuildingSubtype)
	switch {
	case strings.Contains(value, "apartment"), strings.Contains(value, "kerrostalo"), strings.Contains(value, "asunto"):
		return "apartment"
	case strings.Contains(value, "row"), strings.Contains(value, "rivitalo"):
		return "row_house"
	case strings.Contains(value, "detached"), strings.Contains(value, "omakotitalo"):
		return "detached"
	default:
		return ""
	}
}

func nextRenovationDueYear(basisYear int32, lifespanYears int32, startYear int32) int32 {
	dueYear := basisYear + lifespanYears
	for dueYear < startYear {
		dueYear += lifespanYears
	}
	return dueYear
}

func expectedRenovationNeed(rule renovationForecastRule, renovation BuildingRenovation, year int32, basisYear int32, source string) ApartmentRenovationNeed {
	lifecycle := renovationLifecycleForEvidence(rule, renovation)
	windowStart, windowEnd := renovationWindow(year, rule.WindowYears)
	cycleYears := lifecycle.LifespanYears
	scope := lifecycle.Scope
	basis := renovationNeedBasis(rule, renovation, basisYear, source)
	return ApartmentRenovationNeed{Category: rule.Category, Component: renovation.Component, Status: "expected", Scope: scope, Stage: firstNonEmpty(renovation.Stage, "unknown"), Responsibility: firstNonEmpty(renovation.Responsibility, "unknown"), Year: &year, WindowStartYear: &windowStart, WindowEndYear: &windowEnd, BasisYear: &basisYear, CycleYears: &cycleYears, Severity: rule.Severity, Confidence: renovationNeedConfidence(lifecycle, renovation, source), CostEstimateEUR: renovation.CostEstimateEUR, PriceEffect: renovationPriceEffect(rule.Category, "planned"), Source: source, Basis: basis, DependsOn: rule.DependsOn, PriceMechanisms: rule.PriceMechanisms, Explanation: expectedRenovationExplanation(rule, lifecycle, year, basisYear, source)}
}

func renovationLifecycleForEvidence(rule renovationForecastRule, renovation BuildingRenovation) renovationLifecycle {
	scope := firstNonEmpty(renovation.Scope, renovationScopeFromStage(renovation.Stage), inferRenovationScope(renovation.Kind+" "+renovation.Component+" "+renovation.Text), "unknown")
	var fallback renovationLifecycle
	for _, lifecycle := range rule.Lifecycles {
		if lifecycle.Scope == "unknown" {
			fallback = lifecycle
		}
		if lifecycle.Scope == scope {
			return lifecycle
		}
	}
	if fallback.Scope != "" {
		return fallback
	}
	return renovationLifecycle{Scope: "unknown", LifespanYears: 30, Confidence: "low"}
}

func renovationScopeFromStage(stage string) string {
	switch normalizeRenovationStage(stage) {
	case "need_assessment", "project_planning", "tendering":
		return "planning"
	case "condition_survey":
		return "survey"
	case "maintenance":
		return "partial"
	case "execution", "completed":
		return "full"
	default:
		return ""
	}
}

func renovationNeedConfidence(lifecycle renovationLifecycle, renovation BuildingRenovation, source string) string {
	if source == "lifecycle_from_build_year" {
		return "low"
	}
	if renovation.Confidence != nil && *renovation.Confidence >= 80 && lifecycle.Confidence != "" {
		return lifecycle.Confidence
	}
	if renovation.Confidence != nil && *renovation.Confidence < 60 {
		return "low"
	}
	return firstNonEmpty(lifecycle.Confidence, "medium")
}

func renovationNeedBasis(rule renovationForecastRule, renovation BuildingRenovation, basisYear int32, source string) []string {
	if source == "lifecycle_from_build_year" {
		return []string{fmt.Sprintf("building built in %d", basisYear), fmt.Sprintf("%s rule %s", rule.Category, rule.ID)}
	}
	basis := []string{fmt.Sprintf("%s completed in %d", rule.Category, basisYear), fmt.Sprintf("%s rule %s", rule.Category, rule.ID)}
	if renovation.Scope != "" {
		basis = append(basis, "scope "+renovation.Scope)
	}
	if renovation.Stage != "" && renovation.Stage != "unknown" {
		basis = append(basis, "stage "+renovation.Stage)
	}
	if renovation.Component != "" {
		basis = append(basis, "component "+renovation.Component)
	}
	if renovation.Text != "" {
		basis = append(basis, renovation.Text)
	}
	return basis
}

func expectedRenovationExplanation(rule renovationForecastRule, lifecycle renovationLifecycle, year int32, basisYear int32, source string) string {
	if source == "lifecycle_from_build_year" {
		return fmt.Sprintf("%s has no extracted completed renovation, so the forecast uses build year %d and the declarative %s lifecycle of about %d years. The price-relevant window is around %d.", rule.Category, basisYear, lifecycle.Scope, lifecycle.LifespanYears, year)
	}
	return fmt.Sprintf("%s was last completed in %d with %s scope. The declarative %s lifecycle is about %d years, so the next price-relevant window is around %d.", rule.Category, basisYear, lifecycle.Scope, rule.Label, lifecycle.LifespanYears, year)
}

func renovationWindow(year int32, width int32) (int32, int32) {
	if width <= 0 {
		return year, year
	}
	return year - width/2, year + width/2
}

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

func renovationLifecycles(full int32, partial int32, survey int32) []renovationLifecycle {
	return []renovationLifecycle{{Scope: "full", LifespanYears: full, Confidence: "high"}, {Scope: "partial", LifespanYears: partial, Confidence: "medium"}, {Scope: "survey", LifespanYears: survey, Confidence: "medium", CreatesFollowUp: true, FollowUpYears: survey}, {Scope: "planning", LifespanYears: survey + 2, Confidence: "low", CreatesFollowUp: true, FollowUpYears: survey + 2}, {Scope: "unknown", LifespanYears: full, Confidence: "medium"}}
}

func majorRenovationPriceMechanisms() []string {
	return []string{"housing company debt", "maintenance charge pressure", "buyer demand", "debt share uncertainty"}
}

func int32Ptr(value int32) *int32 {
	return &value
}
