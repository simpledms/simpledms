package widget

type formValidationRules []string

func (rules formValidationRules) Contains(search string) bool {
	for _, rule := range rules {
		if rule == search {
			return true
		}
	}

	return false
}
