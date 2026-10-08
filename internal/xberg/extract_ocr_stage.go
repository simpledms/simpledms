package xberg

type extractOCRStage struct {
	Backend         string `json:"backend"`
	Priority        int    `json:"priority"`
	Language        string `json:"language"`
	TesseractConfig struct {
		OutputFormat string `json:"output_format"`
	} `json:"tesseract_config"`
}

func newTesseractTextStage(ocrLanguage string) extractOCRStage {
	stage := extractOCRStage{
		Backend:  "tesseract",
		Priority: 100,
		Language: ocrLanguage,
	}
	stage.TesseractConfig.OutputFormat = "text"
	return stage
}
