package timex

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DateSuggester struct {
	pattern *regexp.Regexp
	timeNow func() time.Time
}

var dateSuggesterRegexp = regexp.MustCompile(`\b(\d{4})[.\-/](\d{1,2})[.\-/](\d{1,2})\b|\b(\d{1,2})[.\-/](\d{1,2})[.\-/](\d{4})\b|\b(\d{1,2})\.(\d{1,2})\.(\d{2})\b`)

func NewDateSuggester() *DateSuggester {
	return &DateSuggester{
		pattern: dateSuggesterRegexp,
		timeNow: TimeNow,
	}
}

func SuggestDatesFromText(content string) []Date {
	return NewDateSuggester().SuggestFromText(content)
}

func (qq *DateSuggester) SuggestFromText(content string) []Date {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	var suggestions []Date
	seen := make(map[string]struct{})
	for _, match := range qq.pattern.FindAllStringSubmatch(content, -1) {
		year, month, day, ok := qq.parseDateMatch(match)
		if !ok {
			continue
		}
		date, ok := qq.normalizeDate(year, month, day)
		if !ok {
			continue
		}
		key := date.Format("2006-01-02")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, date)
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Time.Before(suggestions[j].Time)
	})

	return suggestions
}

func (qq *DateSuggester) parseDateMatch(match []string) (int, int, int, bool) {
	if len(match) < 10 {
		return 0, 0, 0, false
	}

	if match[1] != "" {
		year, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, 0, 0, false
		}
		month, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, 0, 0, false
		}
		day, err := strconv.Atoi(match[3])
		if err != nil {
			return 0, 0, 0, false
		}
		return year, month, day, true
	}

	if match[4] != "" {
		day, err := strconv.Atoi(match[4])
		if err != nil {
			return 0, 0, 0, false
		}
		month, err := strconv.Atoi(match[5])
		if err != nil {
			return 0, 0, 0, false
		}
		// TODO consider tenant country to resolve DMY/MDY ambiguity.
		year, err := strconv.Atoi(match[6])
		if err != nil {
			return 0, 0, 0, false
		}
		return year, month, day, true
	}

	if match[7] != "" {
		day, err := strconv.Atoi(match[7])
		if err != nil {
			return 0, 0, 0, false
		}
		month, err := strconv.Atoi(match[8])
		if err != nil {
			return 0, 0, 0, false
		}
		// TODO consider tenant country and other formats beyond DD.MM.YY.
		year, err := qq.resolveTwoDigitYear(match[9])
		if err != nil {
			return 0, 0, 0, false
		}
		return year, month, day, true
	}

	return 0, 0, 0, false
}

func (qq *DateSuggester) normalizeDate(year int, month int, day int) (Date, bool) {
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return Date{}, false
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return Date{}, false
	}

	return NewDate(date), true
}

func (qq *DateSuggester) resolveTwoDigitYear(raw string) (int, error) {
	yearPart, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if yearPart < 0 || yearPart > 99 {
		return 0, err
	}

	currentYear := qq.timeNow().Year()
	currentCentury := (currentYear / 100) * 100
	candidate := currentCentury + yearPart
	if candidate > currentYear+1 {
		candidate -= 100
	}

	return candidate, nil
}
