package filesource

// Values returns all valid file sources in stable UI order.
func Values() []FileSource {
	return []FileSource{
		WebInterface,
		PWAOSOpen,
		URLImport,
		WebDAV,
		SystemExtraction,
		UnknownLegacy,
	}
}
