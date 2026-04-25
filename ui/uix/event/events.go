package event

import (
	"github.com/marcobeierer/go-core/ui/uix/events"
)

const (
	SuperTagUpdated events.EventWithID = "superTagUpdated"
	TagCreated      events.Event       = "tagCreated"
	TagUpdated      events.Event       = "tagUpdated"
	TagDeleted      events.Event       = "tagDeleted"
	// use TagUpdated for the moment
	// TagMovedToGroup    EventWithID = "tagMovedToGroup"

	FilterTagsChanged         events.Event = "filterTagsChanged"
	DocumentTypeFilterChanged events.Event = "documentTypeFilterChanged"

	DocumentTypeCreated          events.Event = "documentTypeCreated"
	DocumentTypeDeleted          events.Event = "documentTypeDeleted"
	DocumentTypeUpdated          events.Event = "documentTypeUpdated"
	DocumentTypeAttributeCreated events.Event = "documentTypeAttributeCreated" // TODO own namespace / package?
	DocumentTypeAttributeDeleted events.Event = "documentTypeAttributeDeleted"
	DocumentTypeAttributeUpdated events.Event = "documentTypeAttributeUpdated"
	FolderModeToggled            events.Event = "folderModeToggled"
	SearchQueryUpdated           events.Event = "searchQueryUpdated" // TODO more generic for all inputs?

	PropertyFilterChanged events.Event = "propertyFilterChanged"
	PropertyCreated       events.Event = "propertyCreated"
	PropertyUpdated       events.Event = "propertyUpdated"
	PropertyDeleted       events.Event = "propertyDeleted"

	FilePropertyUpdated events.Event = "filePropertyUpdated"

	FileUpdated        events.Event = "fileUpdated"
	FileDeleted        events.Event = "fileDeleted"
	FileRestored       events.Event = "fileRestored"
	FileMoved          events.Event = "fileMoved"
	FileUploaded       events.Event = "fileUploaded" // used in JS, FileUpload widget
	ZIPArchiveUnzipped events.Event = "zipArchiveUnzipped"

	SortByUpdated events.Event = "sortByUpdated"

	SpaceCreated events.Event = "spaceCreated"
	SpaceUpdated events.Event = "spaceUpdated"
	SpaceDeleted events.Event = "spaceDeleted"

	// if something is added, don't forget to adjust Handler implementation

	// TODO separate Command type or implicit via name?

	// CollapseListItem Event = "collapseListItem"

	UserAssignedToSpace     events.Event = "userAssignedToSpace"
	UserUnassignedFromSpace events.Event = "userUnassignedFromSpace"
)
