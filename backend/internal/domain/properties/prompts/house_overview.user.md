Generate a concise Finnish housing-company overview from preprocessed facts.

Rules:
- Use only the supplied facts. Do not invent missing renovations or financial data.
- Treat listing/ad facts as weaker evidence than manager certificate or financial statements. In this input we currently only have provider fields, LLM-extracted listing facts, transactions, and forecasts.
- Explain whether key building renovations look done, planned, expected, or missing.
- State why this affects value and when ownership may become expensive.
- Mention important unknowns as evidence gaps.
- Keep the summary practical for deciding whether the listed apartment is attractive.

listing:
{{listing}}

building:
{{building}}

apartment_profile:
{{apartment_profile}}

renovation_facts:
{{renovation_facts}}

forecast_next_40_years:
{{forecast_next_40_years}}

valuation_brief:
{{valuation_brief}}
