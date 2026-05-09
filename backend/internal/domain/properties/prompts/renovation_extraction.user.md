Extract structured renovation history from Finnish apartment listing text.

Rules:
- Use status "done" for completed renovations and "planned" for upcoming, future, decided, proposed, or later renovations.
- Preserve future renovations even if no year is provided.
- Normalize categories to concise English keys: pipe, sewer, water_supply, facade, roof, window, balcony, electricity, elevator, heating, ventilation, drainage, yard, common_area, bathroom, kitchen, other.
- Also extract component, scope, stage, responsibility, and cost_estimate_eur when supported by the text.
- For Finnish project stages, map kunnossapitotarveselvitys/tarveselvitys to need_assessment, kuntotutkimus/kartoitus/kuvaus to condition_survey, hankesuunnittelu/suunnittelu to project_planning, kilpailutus to tendering, päätös to decision, urakka/toteutus to execution.
- For scope, use survey for kuntotutkimus/kartoitus/kuvaus, planning for tarveselvitys/hankesuunnittelu/suunnittelu/kilpailutus, partial for huolto/osittainen/sukitus/maalaus/lakkaus, full for uusinta/saneeraus/peruskorjaus.
- For responsibility, use housing_company for taloyhtiö/kiinteistö/building systems/common areas, shareholder only when text clearly says apartment/shareholder responsibility, otherwise unknown.
- If a phrase says no planned renovations, do not create an item for it.

renovations_done_text:
{{renovations_done_text}}

renovations_planned_text:
{{renovations_planned_text}}
