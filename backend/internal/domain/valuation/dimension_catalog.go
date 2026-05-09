package valuation

type InputDimension struct {
	Section   string
	Key       string
	ValueKind string
	Promoted  bool
}

var inputDimensionCatalog = map[string]InputDimension{
	"balcony.glazing":                    {Section: "balcony", Key: "glazing", ValueKind: "bool", Promoted: true},
	"balcony.has_balcony":                {Section: "balcony", Key: "has_balcony", ValueKind: "bool", Promoted: true},
	"unit.balcony":                       {Section: "unit", Key: "balcony", ValueKind: "bool", Promoted: true},
	"unit.sauna":                         {Section: "unit", Key: "sauna", ValueKind: "bool", Promoted: true},
	"unit.area_m2":                       {Section: "unit", Key: "area_m2", ValueKind: "number", Promoted: true},
	"unit.living_area_m2":                {Section: "unit", Key: "living_area_m2", ValueKind: "number", Promoted: true},
	"unit.total_area_m2":                 {Section: "unit", Key: "total_area_m2", ValueKind: "number", Promoted: true},
	"unit.other_area_m2":                 {Section: "unit", Key: "other_area_m2", ValueKind: "number", Promoted: true},
	"unit.floor_level":                   {Section: "unit", Key: "floor_level", ValueKind: "number", Promoted: true},
	"unit.accessibility":                 {Section: "unit", Key: "accessibility", ValueKind: "text", Promoted: true},
	"layout.room_layout":                 {Section: "layout", Key: "room_layout", ValueKind: "text", Promoted: true},
	"layout.room_count":                  {Section: "layout", Key: "room_count", ValueKind: "number", Promoted: true},
	"layout.kitchen_type":                {Section: "layout", Key: "kitchen_type", ValueKind: "text", Promoted: true},
	"layout.has_separate_kitchen":        {Section: "layout", Key: "has_separate_kitchen", ValueKind: "bool", Promoted: true},
	"layout.has_open_kitchen":            {Section: "layout", Key: "has_open_kitchen", ValueKind: "bool", Promoted: true},
	"layout.has_alcove":                  {Section: "layout", Key: "has_alcove", ValueKind: "bool", Promoted: true},
	"layout.awkward_layout":              {Section: "layout", Key: "awkward_layout", ValueKind: "bool", Promoted: true},
	"layout.layout_quality":              {Section: "layout", Key: "layout_quality", ValueKind: "text", Promoted: true},
	"layout.separate_wc_count":           {Section: "layout", Key: "separate_wc_count", ValueKind: "number", Promoted: true},
	"rooms.separate_wc_count":            {Section: "rooms", Key: "separate_wc_count", ValueKind: "number", Promoted: true},
	"sauna.has_sauna":                    {Section: "sauna", Key: "has_sauna", ValueKind: "bool", Promoted: true},
	"sauna.private_sauna":                {Section: "sauna", Key: "private_sauna", ValueKind: "bool", Promoted: true},
	"storage.storage_quality":            {Section: "storage", Key: "storage_quality", ValueKind: "text", Promoted: true},
	"views.view_quality":                 {Section: "views", Key: "view_quality", ValueKind: "text", Promoted: true},
	"views.noise_risk":                   {Section: "views", Key: "noise_risk", ValueKind: "bool", Promoted: true},
	"condition.surface_renovation_need":  {Section: "condition", Key: "surface_renovation_need", ValueKind: "bool", Promoted: true},
	"condition.modernization_need":       {Section: "condition", Key: "modernization_need", ValueKind: "bool", Promoted: true},
	"kitchen.renovated":                  {Section: "kitchen", Key: "renovated", ValueKind: "bool", Promoted: true},
	"bathroom.renovated":                 {Section: "bathroom", Key: "renovated", ValueKind: "bool", Promoted: true},
	"heating.heating_method":             {Section: "heating", Key: "heating_method", ValueKind: "text", Promoted: true},
	"building.accessibility":             {Section: "building", Key: "accessibility", ValueKind: "text", Promoted: true},
	"building.common_area_quality":       {Section: "building", Key: "common_area_quality", ValueKind: "text", Promoted: true},
	"building.build_year":                {Section: "building", Key: "build_year", ValueKind: "number", Promoted: true},
	"building.energy_class":              {Section: "building", Key: "energy_class", ValueKind: "text", Promoted: true},
	"building.heating_method":            {Section: "building", Key: "heating_method", ValueKind: "text", Promoted: true},
	"building.material":                  {Section: "building", Key: "material", ValueKind: "text", Promoted: true},
	"building.roof_type":                 {Section: "building", Key: "roof_type", ValueKind: "text", Promoted: true},
	"building.roof_material":             {Section: "building", Key: "roof_material", ValueKind: "text", Promoted: true},
	"building.elevator":                  {Section: "building", Key: "elevator", ValueKind: "bool", Promoted: true},
	"building.apartment_count":           {Section: "building", Key: "apartment_count", ValueKind: "number", Promoted: true},
	"site.plot_ownership_type":           {Section: "site", Key: "plot_ownership_type", ValueKind: "text", Promoted: true},
	"charges.maintenance_charge_monthly": {Section: "charges", Key: "maintenance_charge_monthly", ValueKind: "number", Promoted: true},
	"charges.capital_charge_monthly":     {Section: "charges", Key: "capital_charge_monthly", ValueKind: "number", Promoted: true},
	"charges.total_charge_monthly":       {Section: "charges", Key: "total_charge_monthly", ValueKind: "number", Promoted: true},
	"charges.debt_share_eur":             {Section: "charges", Key: "debt_share_eur", ValueKind: "number", Promoted: true},
	"charges.charge_risk":                {Section: "charges", Key: "charge_risk", ValueKind: "text", Promoted: true},
	"risk.shareholder_liability":         {Section: "risk", Key: "shareholder_liability", ValueKind: "text", Promoted: true},
	"risk.financial_risk":                {Section: "risk", Key: "financial_risk", ValueKind: "text", Promoted: true},
	"risk.maintenance_risk":              {Section: "risk", Key: "maintenance_risk", ValueKind: "text", Promoted: true},
	"risk.repair_backlog_risk":           {Section: "risk", Key: "repair_backlog_risk", ValueKind: "text", Promoted: true},
}

func inputDimensionFor(section string, key string) (InputDimension, bool) {
	dimension, ok := inputDimensionCatalog[section+"."+key]
	return dimension, ok
}
