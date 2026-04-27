package filenamex

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// https://en.wikipedia.org/wiki/Filename
// TODO on Windows there are also some reserved words like CON, CONIN$
var (
	// just \x00 on Linux
	asciiControlChars               = regexp.MustCompile(`[\x00-\x1F\x7F]`)
	disallowedChars_UNIX            = regexp.MustCompile(`/`)
	disallowedChars_VFAT_exFAT_NTFS = regexp.MustCompile(`["*/:<>?\\|]`)
	// disallowedChars_VFAT_exFAT_NTFS = regexp.MustCompile(`[\"\*\/:<>\?\\|]`)
	unicodeWhitespaceBeginOrEnd = regexp.MustCompile(`^[[:space:]].*[[:space:]]$`)

	// from https://en.wikipedia.org/wiki/Filename
	// not 100 percent sure if CONIN$ are also forbidden on modern file systems like NTFS
	windowsAndDOSReservedNames = []string{
		"CON", "CONIN$", "CONOUT$", "PRN",
		"AUX", "CLOCK$", "NUL",
		"COM0", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"COM¹", "COM²", "COM³",
		"LPT0", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		"LPT¹", "LPT²", "LPT³",
		"LST", "KEYBD$", "SCREEN$", "$IDLE$", "CONFIG$",
	}

	// all lower case for easier check below
	ntfsReservedNamesInRoot = []string{
		"$mft", "$mftmirr", "$logfile", "$volume", "$attrdef", "$bitmap", "$boot", "$badclus", "$secure",
		"$upcase", "$extend", "$quota", "$objid", "$reparse",
		// "$Mft", "$MftMirr", "$LogFile", "$Volume", "$AttrDef", "$Bitmap", "$Boot", "$BadClus", "$Secure",
		// "$Upcase", "$Extend", "$Quota", "$ObjId", "$Reparse",
	}
)

// supported file systems are: VFAT, exFAT, NTFS, common UNIX systems
// there are some names reserved in NTFS root directory, like $AttrDef,
// but they are not checked currently
//
// if no Object storage is used, creation on file system with os.Mkdir or os.OpenFile should
// provide some additional safety, but could lead to issues with WebDAV;
//
// if the local file system is used directly (when importing or on a daily basis) it should be
// safe as long as the instance is not migrated to another operating system and if WebDAV
// is only used with the same system
//
// TODO UX: return error message?
func IsAllowed(filename string) bool {
	if !filepath.IsLocal(filename) {
		return false
	}
	if asciiControlChars.MatchString(filename) {
		return false
	}
	if disallowedChars_UNIX.MatchString(filename) {
		return false
	}
	if disallowedChars_VFAT_exFAT_NTFS.MatchString(filename) {
		return false
	}
	if unicodeWhitespaceBeginOrEnd.MatchString(filename) {
		return false
	}
	if slices.Contains(windowsAndDOSReservedNames, strings.ToUpper(filename)) {
		return false
	}
	if slices.Contains(ntfsReservedNamesInRoot, strings.ToLower(filename)) {
		return false
	}
	if len(filename) > 255 {
		return false
	}
	if len(filename) < 1 {
		return false
	}
	if dir, _ := filepath.Split(filename); dir != "" {
		return false
	}
	return true
}
