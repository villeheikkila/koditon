package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"koditon/internal/domain/properties"
)

type ManagerCertificateParseFlags struct {
	OfferingID string
	Path       string
	Model      string
	JSON       bool
	Out        io.Writer
}

type ManagerCertificateService interface {
	UploadManagerCertificate(context.Context, string, properties.PropertyDocumentUpload) (properties.PropertyDocumentSummary, error)
	ExtractManagerCertificate(context.Context, string, string) (properties.ManagerCertificateExtractionResult, error)
	ExtractManagerCertificatePDF(context.Context, properties.PropertyDocumentUpload, string) (properties.ManagerCertificatePDFExtractionResult, error)
	ProjectManagerCertificateExtraction(context.Context, string) (properties.ManagerCertificateExtractionResult, error)
}

func RunManagerCertificateParse(ctx context.Context, svc ManagerCertificateService, f ManagerCertificateParseFlags) error {
	out := resolveOutput(f.Out)
	if f.Path == "" {
		return fmt.Errorf("pdf path is required")
	}
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("read manager certificate pdf: %w", err)
	}
	upload := properties.PropertyDocumentUpload{Filename: filepath.Base(f.Path), MimeType: "application/pdf", Bytes: data}
	if f.OfferingID == "" {
		result, err := svc.ExtractManagerCertificatePDF(ctx, upload, f.Model)
		if err != nil {
			return fmt.Errorf("extract manager certificate pdf: %w", err)
		}
		if f.JSON {
			return WriteJSON(out, result)
		}
		fmt.Fprintln(out, headerStyle.Render("Manager certificate parsed"))
		fmt.Fprintf(out, "Filename: %s\n", result.Filename)
		fmt.Fprintf(out, "Model: %s\n", result.Model)
		fmt.Fprintln(out, string(result.RawJSON))
		return nil
	}
	document, err := svc.UploadManagerCertificate(ctx, f.OfferingID, upload)
	if err != nil {
		return fmt.Errorf("upload manager certificate: %w", err)
	}
	result, err := svc.ExtractManagerCertificate(ctx, document.ID, f.Model)
	if err != nil {
		return fmt.Errorf("extract manager certificate: %w", err)
	}
	if f.JSON {
		return WriteJSON(out, result)
	}
	fmt.Fprintln(out, headerStyle.Render("Manager certificate parsed"))
	fmt.Fprintf(out, "Document: %s\n", result.Document.ID)
	fmt.Fprintf(out, "Offering: %s\n", result.Document.OfferingID)
	fmt.Fprintf(out, "Model: %s\n", result.Model)
	fmt.Fprintf(out, "Claims: %d\n", result.Claims)
	fmt.Fprintf(out, "Status: %s\n", result.Document.ExtractionStatus)
	if result.Document.ExtractionError != "" {
		fmt.Fprintf(out, "Error: %s\n", result.Document.ExtractionError)
	}
	return nil
}

func RunManagerCertificateProject(ctx context.Context, svc ManagerCertificateService, documentID string, jsonOutput bool, out io.Writer) error {
	if documentID == "" {
		return fmt.Errorf("document id is required")
	}
	result, err := svc.ProjectManagerCertificateExtraction(ctx, documentID)
	if err != nil {
		return fmt.Errorf("project manager certificate: %w", err)
	}
	if jsonOutput {
		return WriteJSON(out, result)
	}
	fmt.Fprintln(resolveOutput(out), headerStyle.Render("Manager certificate projected"))
	fmt.Fprintf(resolveOutput(out), "Document: %s\n", result.Document.ID)
	fmt.Fprintf(resolveOutput(out), "Offering: %s\n", result.Document.OfferingID)
	fmt.Fprintf(resolveOutput(out), "Model: %s\n", result.Model)
	fmt.Fprintf(resolveOutput(out), "Claims: %d\n", result.Claims)
	return nil
}
