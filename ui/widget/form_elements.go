package widget

import (
	"fmt"
	"log"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"github.com/marcobeierer/structs"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/simpledms/simpledms/ctxx"
)

// !!!!
// TODO FormElements needs refactoring; was copied over from go-formutils on 13.03.2025 and
//		extended with Widget support; future version should only support widgets; this would
//		simplify the code a lot
// !!!!

// TODO rfc3339 could imply date or datetime
// TODO impl a schema for `form` tag, currently used for `-` and `,flatten`

// TODO rename to fields? it's not just Fields, but also fieldsets, maybe blocks, etc.
type formElements []*formElement

// NewElements uses default config, if you need a custom config, use NewElementsWithConfig instead
func newFormElements(ctx ctxx.Context, obj interface{}) formElements {
	return newFormElementsWithConfig(ctx, obj, newFormConfig())
}

func newFormElementsWithConfig(ctx ctxx.Context, obj interface{}, config *formConfig) formElements {
	if !structs.IsStruct(obj) {
		log.Println("obj is not a struct")
		return formElements{}
	}

	return newElementsFromFields(ctx, structs.Fields(obj), config, "%s", nil, 0, false)
}

// unsafe means that value can be nil
func newElementsFromFields(ctx ctxx.Context, fields []*structs.Field, config *formConfig, nameFormatStr string, unsafeParentField *structs.Field, currentNestingLevel int64, isParentHidden bool) formElements {
	elements := formElements{}

	for _, structField := range fields {
		formTag := structField.Tag("form")
		if formTag == "-" {
			continue
		}

		formParentNameTag := structField.Tag("form_parent_name")
		if formParentNameTag == "1" && unsafeParentField == nil {
			log.Println("parent field was nil, but form_parent_name was set to 1, should be fixed")
			return elements
		}

		defaultValue := structField.Tag("default_value")

		validTag := []string{}
		validTagMap := map[string]bool{}

		attributes := formAttributes{}

		for _, validatorTagName := range []string{"validate", "valid"} {
			if validTagStr := structField.Tag(validatorTagName); validTagStr != "" { // necessary because split() on empty string results in a slice with one element ("")
				for _, tag := range strings.Split(validTagStr, ",") {
					validTagMap[tag] = true
				}
			}
		}

		if formParentNameTag == "1" {
			for _, validatorTagName := range []string{"validate", "valid"} {
				if validTagStr := unsafeParentField.Tag(validatorTagName); validTagStr != "" { // necessary because split() on empty string results in a slice with one element ("")
					for _, tag := range strings.Split(validTagStr, ",") {
						validTagMap[tag] = true
					}
				}
			}

			attrsTag := unsafeParentField.Tag("form_attrs")
			if attrsTag != "" {
				tags := strings.Split(attrsTag, ",")
				for _, tag := range tags {
					elems := strings.Split(tag, "=")
					if len(elems) != 2 {
						log.Println("there were more than 2 elems in form_attrs assignment:", tag)
						continue
					}

					attributes[elems[0]] = elems[1]
				}
			}

		}

		for tag, _ := range validTagMap {
			validTag = append(validTag, tag)
		}

		hasAutofocus := false

		attrsTag := structField.Tag("form_attrs")
		if attrsTag != "" {
			tags := strings.Split(attrsTag, ",")
			for _, tag := range tags {
				elems := strings.Split(tag, "=")
				if len(elems) < 1 || len(elems) > 2 {
					log.Println("there were less than 1 or more than 2 elems in form_attrs assignment:", tag)
					continue
				}

				value := ""
				if len(elems) > 1 {
					value = elems[1]
				}

				attributes[elems[0]] = value

				if elems[0] == "autofocus" {
					hasAutofocus = true
				}
			}
		}

		element := &formElement{
			Name:               structField.Name(),
			Type:               structField.Kind().String(),
			DefaultValue:       defaultValue,
			LeadingIcon:        structField.Tag("form_leading_icon"),
			TrailingIcon:       structField.Tag("form_trailing_icon"),
			Validation:         validTag,
			Children:           formElements{},
			Resource:           structField.Tag("resource"),
			ResourceLabelField: structField.Tag("resource_label_field"),
			Attributes:         attributes,
		}

		if !structField.IsExported() {
			continue
		}

		// !IsZero check is very important for TTCore employee form
		if defaultValue == "" && structField.Value() != nil && !structField.IsZero() {
			// used to set default value if struct already has a value, for example if set in constructor
			// element.DefaultValue = structField.Value()
			element.DefaultValue = fmt.Sprintf("%v", structField.Value())

			// for TextFields widgets
			//
			// as replacement for code below on 01.05.2025, not sure if it works in all cases...
			defaultValue = fmt.Sprintf("%v", structField.Value())
			/*
				// for TextFields widgets
				switch structField.Kind() {
				case reflect.String:
					defaultValue = structField.Value().(string)
				case reflect.Int:
					val, ok := structField.Value().(int)
					if ok {
						defaultValue = fmt.Sprintf("%d", val)
					} else {
						// for example the case for enums, Kind() is int, but cannot be
						// casted to int
						log.Println("not an int, but " + fmt.Sprintf("%T", structField.Value()))
						defaultValue = fmt.Sprintf("%v", structField.Value())
					}
				case reflect.Int64:
					defaultValue = fmt.Sprintf("%d", structField.Value().(int64))
				case reflect.Int32:
					defaultValue = fmt.Sprintf("%d", structField.Value().(int32))
				case reflect.Slice:
				default:
					// FIXME Slice support must be implemented for selects (recursive call?)
					log.Println("not implemented, kind was " + structField.Kind().String() + " value was " + fmt.Sprintf("%v", structField.Value()))
				}
			*/
		}

		hasProtobufTag := structField.Tag("protobuf") != ""
		attrName := structField.Name()

		if hasProtobufTag {
			// works only for ASCII chars, but should be fine because structField.Name() should always be ASCII
			// first char to lowercase:
			runes := []rune(attrName)
			runes[0] = unicode.ToLower(runes[0])
			attrName = string(runes)

		} else if jsonValue := structField.Tag("json"); jsonValue != "" {
			if valueArr := strings.Split(jsonValue, ","); len(valueArr) > 0 && valueArr[0] != "" {
				attrName = valueArr[0]
			}
		}
		attrName = fmt.Sprintf(nameFormatStr, attrName)

		formTagElems := strings.Split(formTag, ",")
		formFlatten := len(formTagElems) == 2 && formTagElems[1] == "flatten"

		attrTypeTag := structField.Tag("form_attr_type")
		if isParentHidden {
			attrTypeTag = "hidden"
		}

		if structField.Kind() == reflect.Ptr && formFlatten {
			embedInFieldset := true

			_, hasValueField := structField.FieldOk("Value")
			if hasValueField {
				exportedFieldsCount := 0
				for _, field := range structField.Fields() {
					formTagElems := strings.Split(field.Tag("form"), ",")
					formIgnored := formTagElems[0] == "-"

					if field.IsExported() && !formIgnored {
						exportedFieldsCount++
					}
				}
				if exportedFieldsCount == 1 {
					embedInFieldset = false // all protobuf value fields like Date or XID
				}
			}

			if embedInFieldset {
				fieldsetAttributes := formAttributes{}

				if (isParentHidden || attrTypeTag == "hidden") && !config.IgnoreHiddenAttrTypeTag {
					// necessary for spacing with (space-y-5 class) in forms
					// without hidden, the gap is too large
					fieldsetAttributes["hidden"] = ""
				}

				elements = append(elements, &formElement{
					Name:       structField.Name(),
					Type:       fmt.Sprintf("%d", currentNestingLevel),
					Element:    "fieldset",
					Validation: validTag,
					Attributes: fieldsetAttributes,
				})
			}

			elements = append(elements, newElementsFromFields(ctx, structField.Fields(), config, attrName+".%s", structField, currentNestingLevel+1, isParentHidden || attrTypeTag == "hidden")...)

			if embedInFieldset {
				elements = append(elements, &formElement{
					Name:       structField.Name(),
					Type:       fmt.Sprintf("%d", currentNestingLevel),
					Element:    "/fieldset",
					Validation: validTag,
				})
			}
			continue // continue to do not add the element to elements itself

		} else if structField.IsEmbedded() {
			structsTagElems := strings.Split(structField.Tag("structs"), ",")
			structsFlatten := len(structsTagElems) == 2 && structsTagElems[1] == "flatten"

			if structsFlatten {
				// if flatten, add the child elements
				elements = append(elements, newFormElements(ctx, structField.Value())...)
				continue // continue to do not add the element to elements itself
			} else {
				// if not flatten add as childs
				element.Element = "div"
				element.Children = append(element.Children, newFormElements(ctx, structField.Value())...)
			}

		} else {
			kind := structField.Kind()
			value := structField.Value()

			typex := reflect.TypeOf(value)

			// TODO impl advanced support, currently just used for Structs and Pointers
			// TODO or maybe even rename tag? just used in protobuf files as of 07.08.2020
			// TODO get rid of form_type and use Validation rule rfc3339date instead?
			formTypeTag := structField.Tag("form_type")

			// TODO check if Elementable

			if kindable, ok := value.(formKindable); ok {
				kind = kindable.Kind()
			}

			// FIXME workaround for protobuf Date message issue
			if formTypeTag == "date" || formTypeTag == "time" || formTypeTag == "datetime" {
				kind = reflect.Struct
			}

			// FIXME is this an acceptable solution for labelling wrapper messages?
			if formParentNameTag == "1" {
				element.Name = unsafeParentField.Name()
				attrTypeTag = unsafeParentField.Tag("form_attr_type")
			}

			// attrTypeTag not set and type is []byte // TODO not 100 percent sure about best order of setting attrTypeTag...
			if attrTypeTag == "" && kind == reflect.Slice && typex.Elem().Kind() == reflect.Uint8 {
				kind = reflect.String
				attrTypeTag = "file"
			}

			element.Attributes["name"] = attrName

			isRequired := element.Validation.Contains("required")
			// alt library would be https://github.com/gobeam/stringy, but already falls apart with
			// Delimited method because it doesn't accept space as delimiter;
			// IMPORTANT duplicate code in extract_form_fields.go
			label := strcase.ToDelimited(element.Name, ' ')
			labelRunes := []rune(label)
			labelRunes[0] = unicode.ToUpper(labelRunes[0])
			label = string(labelRunes)
			if !isRequired && kind != reflect.Bool {
				// TODO translation, but not important, same in DE and EN
				// 		new elements use wx.T() for label
				element.Name = fmt.Sprintf("%s (%s)", T(label).String(ctx), T("optional").String(ctx))
			} else {
				element.Name = T(label).String(ctx)
			}
			// Checkbox uses `label` without (optional) because it doesn't make sense for checkboxes

			var nilableLeadingIcon *Icon
			if element.LeadingIcon != "" {
				nilableLeadingIcon = NewIcon(element.LeadingIcon)
			}

			elementTag := structField.Tag("form_element")
			if elementTag != "" {
				element.Element = elementTag
			} else {
				switch kind {
				case reflect.String:
					attrType := "text"

					if element.Validation.Contains("email") {
						attrType = "email"
					} else if element.Validation.Contains("url") {
						attrType = "url"
					} else if element.Validation.Contains("password") {
						attrType = "password"
					}

					if attrTypeTag != "" {
						attrType = attrTypeTag
					}

					element.Element = "input"
					element.Attributes["type"] = attrType

					element.Widget = &TextField{
						Label:        T(element.Name), // FIXME might also be user defined (properties)...
						Name:         attrName,
						Type:         attrType,
						Step:         "",
						IsRequired:   isRequired,
						LeadingIcon:  nilableLeadingIcon,
						DefaultValue: defaultValue,
						HasAutofocus: hasAutofocus,
						// Attributes:   element.Attributes,
						// Validation:   element.Validation,
					}
				case reflect.Int, reflect.Int64, reflect.Int32:
					if typex.String() == "int" || typex.String() == "int64" || typex.String() == "int32" {
						attrType := "number"

						if attrTypeTag != "" {
							attrType = attrTypeTag
						}

						element.Element = "input"
						element.Attributes["type"] = attrType
						element.Attributes["step"] = "1"

						element.Widget = &TextField{
							Label:        T(element.Name), // FIXME might also be user defined (properties)...
							Name:         attrName,
							Type:         attrType,
							Step:         "1",
							LeadingIcon:  nilableLeadingIcon,
							DefaultValue: defaultValue,
							// Attributes:   element.Attributes,
							// Validation:   element.Validation,
						}
					} else { // enum
						allowedValuesString := structField.Tag("form_allowed_values")
						allowedValues := []string{}

						// necessary because if Split is used with empty string, resulting slice has not length 0, but contains an empty string as first (and only) element
						if allowedValuesString != "" {
							allowedValues = strings.Split(allowedValuesString, ",")
						}

						enum(ctx, element, kind, value, typex, (isParentHidden || attrTypeTag == "hidden") && !config.IgnoreHiddenAttrTypeTag, allowedValues)
					}
				case reflect.Float32, reflect.Float64:
					element.Element = "input"
					element.Attributes["type"] = "number"
					element.Attributes["step"] = "0.01" // TODO should be more flexible

					element.Widget = &TextField{
						Label:        T(element.Name), // FIXME might also be user defined (properties)...
						Name:         attrName,
						Type:         "number",
						Step:         "0.01",
						LeadingIcon:  nilableLeadingIcon,
						DefaultValue: defaultValue,
						// Attributes:   element.Attributes,
						// Validation:   element.Validation,
					}
				case reflect.Bool:
					labelx := T(label)
					if isRequired {
						labelx = Tf("%s (%s)", labelx.String(ctx), T("required").String(ctx))
					}

					// TODO replace with Switch
					element.Widget = &Checkbox{
						// use T(label) and not element.Name because element.Name is already translated
						Label:      labelx,
						Name:       attrName,
						Value:      "true",
						IsChecked:  value.(bool),
						IsRequired: isRequired,
					}
				case reflect.Struct, reflect.Ptr:
					typexString := typex.String()
					if formTypeTag != "" {
						typexString = formTypeTag
					}

					switch typexString {
					case "datetime", "time.Time", "timex.DateTime", "*timestamppb.Timestamp":
						element.Element = "input"
						element.Attributes["type"] = "datetime-local" // NOTE not widely supported in browsers, currently airpicker or custom element are used instead

					case "date", "timex.Date", "*commonpb.Date", "*personpb.Date": // TODO remove personpb as soon as moved to commonpb...
						element.Element = "input"
						element.Attributes["type"] = "date"

					case "*modelpb.DateRange", "dbutil.DateRange":
						element.Element = "input"
						element.Attributes["type"] = "daterange"

					case "*modelpb.DateTimeRange", "dbutil.DateTimeRange":
						element.Element = "input"
						element.Attributes["type"] = "date-time-range"

					case "time", "timex.Time": // Desktop Safari added support to technical preview on 22 October 2020, thus should be in Safari 14.1 or 15
						element.Element = "input"
						element.Attributes["type"] = "time"

					case "dbutil.NullForeignKey":
						element.Type = "int64"
						element.Element = "input"
						element.Attributes["type"] = "number"

					default:
						// FIXME workaround for proto ID fields like CountryID, etc.
						element.Attributes["name"] = attrName + ".value"

						log.Println("reflect.Struct", typex.String(), "not supported")
					}

					isHidden := (isParentHidden || attrTypeTag == "hidden") && !config.IgnoreHiddenAttrTypeTag
					if isHidden {
						element.Element = "input"
						element.Attributes["type"] = "hidden"
						element.Resource = "" // otherwise it would be handled as select field...
					}

				case reflect.Map:
					switch typex.String() {
					// case "websitesmodel.Params": // TODO impl something more generic // use an interface?
					// params(element, kind, value, typex.Key())
					default:
						log.Println("reflect.Map", typex.String(), "not supported")
					}
				default:
					log.Printf("no rule for kind %s, field %s defined\n", kind.String(), structField.Name())
				}
			}
		}

		elements = append(elements, element)
	}

	return elements

}

func params(element *formElement, kind reflect.Kind, value interface{}, typex reflect.Type) {
	element.Element = "div"
	elemx := reflect.New(typex).Elem()

	for qi := 1; qi < 100; qi++ { // TODO while()
		elemx.SetInt(int64(qi))

		text, ok := elemx.Interface().(fmt.Stringer)
		if !ok {
			log.Printf("element of type %s does not implement fmt.Stringer", typex.String())
			continue
		}

		typeWithoutPackageName := typex.String()[strings.IndexRune(typex.String(), '.')+1:]

		if text.String() == fmt.Sprintf("%s(%d)", typeWithoutPackageName, qi) {
			break
		}

		element.Children = append(element.Children, &formElement{
			Element: "input",
			Attributes: formAttributes{
				"type": "text",
				"name": fmt.Sprintf("Params[%s]", text.String()),
			},
		})
	}
}

func enum(ctx ctxx.Context, element *formElement, kind reflect.Kind, value interface{}, typex reflect.Type, isHidden bool, allowedValues []string) {
	if isHidden {
		element.Element = "input"
		element.Attributes["type"] = "hidden"
		return
	}

	element.Element = "select"
	element.Type = "enum"            // otherwise it would be int...
	element.RawType = typex.String() // for example genderpb.Gender

	hasNamedNullValue := true

	elemx := reflect.New(typex).Elem()

	if !element.Validation.Contains("required") || (element.Validation.Contains("required") && element.DefaultValue == "") {
		element.Children = append(element.Children, &formElement{
			Name:    "-", // TODO something like `- Please select -`?
			Element: "option",
			Attributes: formAttributes{
				"value": "", // TODO better message?
			},
		})
	}

	for qi := 0; qi < 100; qi++ { // TODO while()
		elemx.SetInt(int64(qi))

		text, ok := elemx.Interface().(fmt.Stringer)
		if !ok {
			// log.Println("element of type %s does not implement fmt.Stringer", typex.String())
			continue
		}

		typeWithoutPackageName := typex.String()[strings.IndexRune(typex.String(), '.')+1:]

		// Unknown is a special case for unnamed null values and used hardcoded in some places
		if text.String() == fmt.Sprintf("%s(%d)", typeWithoutPackageName, qi) ||
			text.String() == fmt.Sprintf("%d", qi) || text.String() == "Unknown" {
			if qi == 0 {
				// skip if first, because it has no named default value
				hasNamedNullValue = false
				continue
			}

			if text.String() != "Unknown" {
				// if not first elem, then it is the first invalid
				break
			}
		}

		value := text.String()
		if valuer, ok := elemx.Interface().(formValuer); ok {
			value = valuer.FormValue()
		}

		if len(allowedValues) > 0 {
			found := false

			for _, val := range allowedValues {
				if value == val {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}

		// if null value but not named, it `continues` above where hasNamedNullValue is set to false
		isNamedNullValue := qi == 0

		// don't add named null values if required, but add named null value is required-as-dependency
		if element.Validation.Contains("required") && isNamedNullValue {
			continue // do not add named null values if required
		}

		child := &formElement{
			Name:    T(text.String()).String(ctx),
			Element: "option",
			Attributes: formAttributes{
				// "value": strconv.Itoa(qi),
				"value": value,
			},
		}

		if value == element.DefaultValue {
			child.Attributes["selected"] = "selected"
		}

		element.Children = append(element.Children, child)
	}

	// English seems to work fine for German umlaute, not sure what the best value would be...
	// TODO ideally set by system or user language
	collatex := collate.New(language.English)
	slices.SortFunc(element.Children, func(i, j *formElement) int {
		// collate is necessary because strings.Compare cannot handle umlaute like Ö and always
		// puts them at the end, after other letters like S, X, etc.
		return collatex.CompareString(i.Name, j.Name)
	})

	if hasNamedNullValue && !element.Validation.Contains("required") {
		// set required to hide first (no value) option of rendered select field (and show named null value instead)
		element.Validation = append(element.Validation, "required")
	}
}
