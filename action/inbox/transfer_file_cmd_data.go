package inbox

type TransferFileCmdData struct {
	FileID             string `validate:"required"`
	DestinationSpaceID string `validate:"required"`
	Message            string
}
