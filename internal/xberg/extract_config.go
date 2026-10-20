package xberg

// extractConfig requests layout-aware markdown so that tables survive, including tables in
// scans, which need layout detection, the TATR table model, and PaddleOCR (Xberg 1.3.6).
// PaddleOCR is set explicitly because Xberg's default Tesseract backend drops lines whose
// words are mostly not in its dictionary, for example names and reference numbers. Its
// `latin` recognition model covers English, German, French, Italian, and Spanish.
//
// Xberg downloads the models from Hugging Face on first use into its cache directory, which
// must be persistent so that they are not downloaded again on every start.
type extractConfig struct {
	OutputFormat         string `json:"output_format"`
	UseLayoutForMarkdown bool   `json:"use_layout_for_markdown"`
	Layout               struct {
		TableModel string `json:"table_model"`
	} `json:"layout"`
	OCR struct {
		Backend  string `json:"backend"`
		Language string `json:"language"`
	} `json:"ocr"`
}

func newExtractConfig() extractConfig {
	config := extractConfig{
		OutputFormat:         "markdown",
		UseLayoutForMarkdown: true,
	}
	config.Layout.TableModel = "tatr"
	config.OCR.Backend = "paddle-ocr"
	config.OCR.Language = "latin"
	return config
}
