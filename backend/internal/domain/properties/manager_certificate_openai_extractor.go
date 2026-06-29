package properties

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"koditon/internal/db"
	"koditon/internal/platform/telemetry"
)

const (
	defaultOpenAIManagerCertificateModel = "gpt-5"
	openAIAPIBaseURL                     = "https://api.openai.com/v1"
)

type openAIManagerCertificateExtractor struct {
	apiKey string
	client *http.Client
}

type openAIFileUploadResponse struct {
	ID string `json:"id"`
}

type openAIResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (e openAIManagerCertificateExtractor) Extract(ctx context.Context, document db.GetPropertyDocumentForExtractionRow, operation propertyLLMOperation, modelName string) (managerCertificateObject, []byte, error) {
	return e.ExtractPDF(ctx, document.PropertyDocumentFilename, document.PropertyDocumentBytes, operation, modelName)
}

func (e openAIManagerCertificateExtractor) ExtractPDF(ctx context.Context, filename string, data []byte, operation propertyLLMOperation, modelName string) (managerCertificateObject, []byte, error) {
	systemPrompt, userPrompt, err := propertyLLMPromptText("manager_certificate_extraction", nil)
	if err != nil {
		return managerCertificateObject{}, nil, err
	}
	fileID, err := e.uploadPDF(ctx, filename, data)
	if err != nil {
		return managerCertificateObject{}, nil, err
	}
	defer e.deleteFile(context.WithoutCancel(ctx), fileID)
	text, err := e.createResponse(ctx, openAIResponseRequest{Model: modelName, Input: []openAIInputMessage{
		{Role: "system", Content: []openAIContentPart{{Type: "input_text", Text: systemPrompt}}},
		{Role: "user", Content: []openAIContentPart{{Type: "input_file", FileID: fileID}, {Type: "input_text", Text: userPrompt}}},
	}, Text: openAITextConfig{Format: openAITextFormat{Type: "json_schema", Name: operation.SchemaName, Description: operation.SchemaDescription, Schema: managerCertificateJSONSchema()}}, MaxOutputTokens: operation.MaxOutputTokens})
	if err != nil {
		return managerCertificateObject{}, nil, err
	}
	var extracted managerCertificateObject
	if err := json.Unmarshal([]byte(text), &extracted); err != nil {
		return managerCertificateObject{}, nil, fmt.Errorf("decode manager certificate response json: %w", err)
	}
	rawJSON, err := json.Marshal(extracted)
	if err != nil {
		return managerCertificateObject{}, nil, fmt.Errorf("marshal manager certificate extraction: %w", err)
	}
	return extracted, rawJSON, nil
}

func (e openAIManagerCertificateExtractor) uploadPDF(ctx context.Context, filename string, data []byte) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("purpose", "user_data"); err != nil {
		return "", fmt.Errorf("write openai file purpose: %w", err)
	}
	part, err := writer.CreateFormFile("file", firstNonEmpty(filename, "isannoitsijantodistus.pdf"))
	if err != nil {
		return "", fmt.Errorf("create openai file part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("write openai file bytes: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close openai file upload body: %w", err)
	}
	var response openAIFileUploadResponse
	if err := e.doJSON(ctx, http.MethodPost, openAIAPIBaseURL+"/files", &body, writer.FormDataContentType(), &response); err != nil {
		return "", fmt.Errorf("upload manager certificate pdf to openai: %w", err)
	}
	if strings.TrimSpace(response.ID) == "" {
		return "", fmt.Errorf("openai file upload returned empty file id")
	}
	return response.ID, nil
}

func (e openAIManagerCertificateExtractor) createResponse(ctx context.Context, request openAIResponseRequest) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("marshal openai response request: %w", err)
	}
	var response openAIResponse
	if err := e.doJSON(ctx, http.MethodPost, openAIAPIBaseURL+"/responses", bytes.NewReader(body), "application/json", &response); err != nil {
		return "", fmt.Errorf("create openai manager certificate response: %w", err)
	}
	text := strings.TrimSpace(response.OutputText)
	if text != "" {
		return text, nil
	}
	for _, output := range response.Output {
		if output.Type != "message" {
			continue
		}
		for _, content := range output.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return strings.TrimSpace(content.Text), nil
			}
		}
	}
	return "", fmt.Errorf("openai response did not contain output text")
}

func (e openAIManagerCertificateExtractor) deleteFile(ctx context.Context, fileID string) {
	_ = e.doJSON(ctx, http.MethodDelete, openAIAPIBaseURL+"/files/"+fileID, nil, "", nil)
}

func (e openAIManagerCertificateExtractor) doJSON(ctx context.Context, method string, url string, body io.Reader, contentType string, out any) (err error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := e.client
	if client == nil {
		client = telemetry.HTTPClient(0, nil)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close openai response body: %w", closeErr)
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("openai %s %s failed with status %d: %s", method, url, resp.StatusCode, strings.TrimSpace(string(errBody)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode openai response: %w", err)
	}
	return nil
}

type openAIResponseRequest struct {
	Model           string               `json:"model"`
	Input           []openAIInputMessage `json:"input"`
	Text            openAITextConfig     `json:"text"`
	MaxOutputTokens int64                `json:"max_output_tokens"`
}

type openAIInputMessage struct {
	Role    string              `json:"role"`
	Content []openAIContentPart `json:"content"`
}

type openAIContentPart struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	FileID string `json:"file_id,omitempty"`
}

type openAITextConfig struct {
	Format openAITextFormat `json:"format"`
}

type openAITextFormat struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

func managerCertificateJSONSchema() map[string]any {
	return obj(map[string]any{
		"document": obj(map[string]any{
			"document_date":    str(),
			"issuer":           str(),
			"property_manager": str(),
			"warnings":         arr(str()),
			"evidence":         evidenceSchema(),
		}),
		"housing_company": obj(map[string]any{
			"name":                str(),
			"business_id":         str(),
			"build_year":          nullableInt(),
			"apartment_count":     nullableInt(),
			"plot_ownership_type": enum("owned", "rented", "unknown", ""),
			"energy_class":        str(),
			"evidence":            evidenceSchema(),
		}),
		"building": obj(map[string]any{
			"build_year":      nullableInt(),
			"floor_count":     nullableInt(),
			"apartment_count": nullableInt(),
			"energy_class":    str(),
			"heating_method":  str(),
			"material":        str(),
			"roof_type":       str(),
			"roof_material":   str(),
			"elevator":        map[string]any{"type": []string{"boolean", "null"}},
			"evidence":        evidenceSchema(),
		}),
		"unit": obj(map[string]any{
			"apartment_number":           str(),
			"shares":                     str(),
			"area_m2":                    nullableNumber(),
			"room_layout":                str(),
			"floor_level":                nullableInt(),
			"maintenance_charge_monthly": nullableNumber(),
			"capital_charge_monthly":     nullableNumber(),
			"total_charge_monthly":       nullableNumber(),
			"debt_share_eur":             nullableNumber(),
			"shareholder_liability":      str(),
			"evidence":                   evidenceSchema(),
		}),
		"finances": obj(map[string]any{
			"financial_risk":      risk(),
			"maintenance_risk":    risk(),
			"repair_backlog_risk": risk(),
			"loan_summary":        str(),
			"charge_summary":      str(),
			"loans":               arr(loanSchema()),
			"charges":             arr(chargeSchema()),
			"evidence":            evidenceSchema(),
		}),
		"renovations": arr(obj(map[string]any{
			"system_type":       enum("pipe", "water_supply", "sewer", "roof", "facade", "window", "balcony", "elevator", "heating", "ventilation", "drainage", "electricity", "yard", "common_areas", "other"),
			"action":            enum("replacement", "repair", "renovation", "maintenance", "inspection", "condition_assessment", "planning", "installation", "painting", "cleaning", "unknown"),
			"source_label":      str(),
			"status":            enum("done", "planned", "suspected", "forecast", "unknown"),
			"stage":             enum("unknown", "study", "condition_assessment", "planning", "tendering", "execution", "completed"),
			"scope":             enum("unknown", "full", "partial", "maintenance"),
			"responsibility":    enum("unknown", "housing_company", "shareholder", "mixed"),
			"year":              nullableInt(),
			"start_year":        nullableInt(),
			"end_year":          nullableInt(),
			"cost_estimate_eur": nullableInt(),
			"summary":           str(),
			"evidence":          evidenceSchema(),
		})),
		"risks": obj(map[string]any{
			"administrative_legal_risk": risk(),
			"restrictions":              arr(str()),
			"disputes":                  arr(str()),
			"missing_evidence":          arr(str()),
			"evidence":                  evidenceSchema(),
		}),
	})
}

func evidenceSchema() map[string]any {
	return arr(obj(map[string]any{"text": str(), "page": nullableInt(), "section": str()}))
}

func loanSchema() map[string]any {
	return obj(map[string]any{
		"name":        str(),
		"lender":      str(),
		"purpose":     str(),
		"balance_eur": nullableNumber(),
		"limit_eur":   nullableNumber(),
		"used_eur":    nullableNumber(),
		"as_of":       str(),
		"evidence":    evidenceSchema(),
	})
}

func chargeSchema() map[string]any {
	return obj(map[string]any{
		"charge_type":    enum("maintenance", "capital", "water", "parking", "sauna", "storage", "other", "unknown"),
		"target":         enum("unit", "housing_company", "unknown"),
		"label":          str(),
		"amount_monthly": nullableNumber(),
		"amount_per_m2":  nullableNumber(),
		"basis":          str(),
		"loan_name":      str(),
		"vat_included":   map[string]any{"type": []string{"boolean", "null"}},
		"evidence":       evidenceSchema(),
	})
}

func obj(properties map[string]any) map[string]any {
	required := make([]string, 0, len(properties))
	for key := range properties {
		required = append(required, key)
	}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}

func arr(items map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

func str() map[string]any {
	return map[string]any{"type": "string"}
}

func integer() map[string]any {
	return map[string]any{"type": "integer"}
}

func nullableInt() map[string]any {
	return map[string]any{"type": []string{"integer", "null"}}
}

func nullableNumber() map[string]any {
	return map[string]any{"type": []string{"number", "null"}}
}

func enum(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func risk() map[string]any {
	return enum("unknown", "low", "medium", "high", "")
}
