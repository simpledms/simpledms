package xberg

// extractConfig requests markdown, which keeps tables that plain output writes cell by cell,
// and pins OCR to a single Tesseract stage with plain text output (Xberg 1.3.6):
//   - Without an explicit pipeline, Xberg adds a PaddleOCR fallback stage, which downloads
//     models from Hugging Face on first use and only recognizes one language per run.
//   - Tesseract's default markdown output runs a non-configurable dictionary filter that
//     drops lines dominated by unknown words, for example names and reference numbers. The
//     document-level markdown output does not.
type extractConfig struct {
	OutputFormat string `json:"output_format"`
	OCR          struct {
		Language string `json:"language"`
		Pipeline struct {
			Stages []extractOCRStage `json:"stages"`
		} `json:"pipeline"`
	} `json:"ocr"`
}

// newExtractConfig expects Tesseract language codes joined by `+`, for example `eng+deu`.
func newExtractConfig(ocrLanguage string) extractConfig {
	var config extractConfig
	config.OutputFormat = "markdown"
	config.OCR.Language = ocrLanguage
	config.OCR.Pipeline.Stages = []extractOCRStage{
		newTesseractTextStage(ocrLanguage),
	}
	return config
}
