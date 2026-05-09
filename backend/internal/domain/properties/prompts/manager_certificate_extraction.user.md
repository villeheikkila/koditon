Extract the relevant facts from this Finnish isännöitsijäntodistus PDF.

Rules:
- Use only facts visible in the PDF.
- Normalize risk levels as low, medium, high, or unknown.
- Normalize plot ownership as owned, rented, or unknown.
- For renovations, include completed repairs, planned repairs, condition assessments, PTS items, and major maintenance decisions.
- Renovation system_type must be one of: pipe, water_supply, sewer, roof, facade, window, balcony, elevator, heating, ventilation, drainage, electricity, yard, common_areas, other.
- Renovation action must be one of: replacement, repair, renovation, maintenance, inspection, condition_assessment, planning, installation, painting, cleaning, unknown.
- Preserve the original Finnish source wording in source_label.
- Capture charges as structured charge rows when visible.
- Capture company loans and credit facilities as structured loan rows when visible.
- Preserve years and cost estimates when present.
- Evidence should include concise text, page number when visible, and source section when recognizable.
- Do not include confidence values.
