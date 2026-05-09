Neighborhood: {{neighborhood}}
Municipality: {{municipality}}

Available postal codes in this municipality:
{{postal_codes_json}}

Instructions:
1. Analyze the neighborhood name and find the best matching postal code area.
2. Consider that neighborhood names might be informal, compound, comma-separated, or partial matches.
3. If no reasonable match exists, return null for postal_code_id.
4. Respond only with valid JSON in this exact format:

{
  "postal_code_id": "uuid-here-or-null",
  "postal_code_name": "name-here-or-empty",
  "confidence": "high|medium|low|none",
  "reasoning": "brief explanation"
}
