package inbox

type ConsumeUploadsCmdData struct {
	UploadToken string `validate:"required"`
}
