# Production Findings

## Bugs

- [ ] if an open file is deleted in Browse view, details are not closed...
- [ ] http: superfluous response.WriteHeader call from github.com/gorilla/handlers.(*compressResponseWriter).WriteHeader (compress.go:26)
  - when encryption is disabled
- [ ] when a metadata text fields gets updated (currently a 1000ms delay is implemented) the field loses focus. It must keep the focus to not interrupt the user.

## UX

- [ ] On tag creation, tag type "Simple" should be preselected.

