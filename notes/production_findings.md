# Production Findings

## Bugs

- [ ] height of multi-line tag groups in filters is not correct; probably set NoOverflowHidden
- [ ] form requests of commands (via FormHelper) are probably not read-only yet (needs verification)
- [ ] when opening a second file with the PWA on Desktop, it doesn't upload the file. The first one works fine.
- [ ] if an open file is deleted in Browse view, details are not closed...
- [ ] http: superfluous response.WriteHeader call from github.com/gorilla/handlers.(*compressResponseWriter).WriteHeader (compress.go:26)
  - when encryption is disabled
- [x] when a metadata text fields gets updated (currently a 1000ms delay is implemented) the field loses focus. It must keep the focus to not interrupt the user.
- [ ] clicking twice on the "Tags" tab in the Browse view details renders the "Create new tag or group" button twice, one is not functional.

## UX

- [ ] On tag creation, tag type "Simple" should be preselected.

## Technical
- [ ] partial package
	- rename to widget? or move to common
		- suffix widget?
