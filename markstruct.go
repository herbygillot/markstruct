// Package markstruct converts a struct's string fields from Markdown to HTML
// in-place.
//
// markstruct will take a pointer to a struct, and scan for tagged string fields,
// then render the value of these fields as Markdown to HTML in-place. That is to
// say the value of each field itself will be changed within the struct to be the HTML
// result of rendering the original field's value as Markdown.  markstruct targets
// fields whose type are string, pointer to string, string slice, and maps with
// string values.  markstruct uses `github.com/yuin/goldmark` to render
// Markdown, and allows for custom goldmark.Markdown objects and parse options.
//
// Fields within a struct that should be converted should be annotated with the
// tag `markdown:"on"`
//
// Example:
//
//  type Document struct {
//    Title string                 // this field will be ignored
//    Body  string `markdown:"on"` // this field will be converted
//  }
//
//   doc := &Document{
// 	  Title: "Doc *1*",
// 	  Body:  "This is _emphasis_.",
//   }
//
//   changed, err := markstruct.ConvertFields(doc)
//   ...
//   fmt.Println(doc.Title) // "Doc *1*"
//   fmt.Println(doc.Body)  // "<p>This is <em>emphasis</em>.</p>"
//
// markstruct can optionally modify all struct string fields unequivocally,
// ignoring the presence of this tag.
package markstruct

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
)

const (
	structTagKey = "markdown"
)

// FieldConverter converts the content of string (and other string-related type)
// fields within a struct from Markdown to HTML in-place.
type FieldConverter interface {
	ConvertFields(s interface{}, opts ...parser.ParseOption) (bool, error)

	ConvertAllFields(s interface{}, opts ...parser.ParseOption) (bool, error)

	ValidateFields(s interface{}, opts ...parser.ParseOption) (bool, error)

	ValidateAllFields(s interface{}, opts ...parser.ParseOption) (bool, error)
}

type converter struct {
	markdown goldmark.Markdown
}

type fieldProcessor struct {
	ConvertAllFields bool
	ValidateOnly     bool

	converter    *converter
	parseOptions []parser.ParseOption
}

var _ FieldConverter = (*converter)(nil)

var defaultConverter = WithMarkdown(goldmark.New())

var (
	// ErrInvalidType signifies that we have received a value of type other
	// than the expected pointer to struct.
	ErrInvalidType = errors.New("invalid type")
)

// ConvertFields accepts a pointer to a struct, and will modify tagged
// fields of relevant type within the struct by replacing each field with its
// contents rendered from Markdown to HTML. ConvertFields returns a boolean
// signifying whether changes were made to the struct or not, as well as
// any error encountered. If given any other type besides a pointer to a
// struct, ConvertFields will return an ErrInvalidType error.
//
// ConvertFields optionally accepts ParseOptions, which are passed to
// `goldmark` to modify Markdown parsing during conversion.
//
// Fields that should be converted within the struct should be tagged with
// `markdown:"on"`:
//
//  type Document struct {
//    Title string
//    Body  string `markdown:"on"`
//  }
//
// ConvertFields supports struct fields of type string, *string, []string,
// and maps with string values.
func ConvertFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return defaultConverter.ConvertFields(s, opts...)
}

// ValidateFields will, like ConvertFields, accept a pointer to a struct
// whose string fields are expected to be tagged with `markdown:"on"`. Unlike
// ConvertFields, ValidateFields makes no changes to the struct or its
// fields.  ValidateFields returns the same values as ConvertFields:
// a boolean indicating whether fields would have been changed, as well
// as any error encountered.
func ValidateFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return defaultConverter.ValidateFields(s, opts...)
}

// ConvertAllFields does the same as ConvertFields, except it will convert
// all fields of relevant type within a struct, regardless of whether the
// field is tagged with `markdown:"on"` or not.  ConvertAllFields also
// optionally accepts `goldmark` ParseOptions which are used to modify
// Markdown parsing during conversion.
//
// Just like ConvertFields, ConvertAllFields only accepts a pointer to a
// struct, and returns a boolean signifying whether the struct was changed,
// as well as any error encountered.
//
// Passing a value of any type other than a pointer to struct will cause
// ConvertAllFields to return an ErrInvalidType error.
func ConvertAllFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return defaultConverter.ConvertAllFields(s, opts...)
}

// ValidateAllFields behaves like ConvertAllFields, except that it makes no
// changes to a struct.  ValidateAllFields can be used to test for errors
// in a situation where ConvertAllFields would be used.  ValidateAllFields
// returns the same return values as ConvertAllFields.
func ValidateAllFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return defaultConverter.ValidateAllFields(s, opts...)
}

// WithMarkdown creates a FieldConverter from a custom `goldmark.Markdown` object.
// Use this with `goldmark.New` to allow using markstruct with non-default `goldmark`
// extensions or configuration.
func WithMarkdown(md goldmark.Markdown) FieldConverter {
	return &converter{
		markdown: md,
	}
}

func (c *converter) ConvertFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return c.process(s, false, false, opts...)
}

func (c *converter) ConvertAllFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return c.process(s, true, false, opts...)
}

func (c *converter) ValidateFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return c.process(s, false, true, opts...)
}

func (c *converter) ValidateAllFields(s interface{}, opts ...parser.ParseOption) (bool, error) {
	return c.process(s, true, true, opts...)
}

func (c *converter) process(s interface{}, allFields bool, validateOnly bool, opts ...parser.ParseOption) (bool, error) {
	objval := reflect.ValueOf(s)

	if !objval.IsValid() {
		return false, nil
	}

	objtype := reflect.TypeOf(s)
	if objtype.Kind() != reflect.Ptr {
		return false, fmt.Errorf("%w: expect pointer to struct", ErrInvalidType)
	}

	elem := objval.Elem()
	if !isValidSettable(elem) {
		return false, nil
	}

	fieldproc := makeFieldProcessor(c, opts...)
	fieldproc.ConvertAllFields = allFields
	fieldproc.ValidateOnly = validateOnly

	return fieldproc.convertStruct(elem)
}

func (f *fieldProcessor) convert(v reflect.Value) (bool, error) {
	switch v.Kind() {
	case reflect.Ptr:
		elem := v.Elem()
		return f.convert(elem)
	case reflect.Slice, reflect.Array:
		return f.convertSlice(v)
	case reflect.Map:
		return f.convertMap(v)
	case reflect.Struct:
		return f.convertStruct(v)
	case reflect.String:
		return f.convertString(v)
	}

	return false, nil
}

func (f *fieldProcessor) convertMap(v reflect.Value) (bool, error) {
	if v.Kind() != reflect.Map {
		return false, fmt.Errorf("%w: expect map", ErrInvalidType)
	}

	// only process maps with string values
	if v.Type().Elem().Kind() != reflect.String {
		return false, nil
	}

	if !v.CanSet() {
		return false, nil
	}

	var changed bool
	var err error

	for _, kval := range v.MapKeys() {
		value := v.MapIndex(kval)

		rawstr := value.String()
		mdstr, err := f.renderString(rawstr)
		if err != nil {
			break
		}

		if rawstr != mdstr {
			if !f.ValidateOnly {
				v.SetMapIndex(kval, reflect.ValueOf(mdstr))
			}

			changed = true
		}
	}

	return changed, err
}

func (f *fieldProcessor) convertSlice(v reflect.Value) (bool, error) {
	if v.Kind() != reflect.Slice {
		return false, fmt.Errorf("%w: expect string slice", ErrInvalidType)
	}

	var changed bool
	var err error

	for i := 0; i < v.Len(); i++ {
		entry := v.Index(i)
		fchanged, err := f.convert(entry)

		changed = fchanged || changed
		if err != nil {
			break
		}
	}

	return changed, err
}

func (f *fieldProcessor) convertStruct(v reflect.Value) (bool, error) {
	if v.Kind() != reflect.Struct {
		return false, fmt.Errorf("%w: expect struct", ErrInvalidType)
	}

	var changed bool
	var err error

	for i := 0; i < v.NumField(); i++ {
		fchanged := false
		field := v.Field(i)

		if !isStruct(field) {
			if !f.ConvertAllFields && !isStructFieldTagEnabled(v, i) {
				continue
			}
		}

		fchanged, err = f.convert(field)
		changed = fchanged || changed

		if err != nil {
			break
		}
	}

	return changed, err
}

func (f *fieldProcessor) convertString(v reflect.Value) (bool, error) {
	if !isValidSettable(v) {
		return false, nil
	}

	value := v.String()
	rendered, err := f.renderString(value)
	if err != nil {
		return false, err
	}

	if !f.ValidateOnly {
		v.SetString(rendered)
	}

	return value != rendered, err
}

func (f *fieldProcessor) renderString(s string) (string, error) {
	b := &strings.Builder{}
	err := f.writeMarkdown([]byte(s), b)
	return b.String(), err
}

func (f *fieldProcessor) writeMarkdown(source []byte, w io.Writer) error {
	return f.converter.markdown.Convert(source, w, f.parseOptions...)
}

func isStruct(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	if v.Kind() == reflect.Struct {
		return true
	}

	if v.Kind() == reflect.Ptr {
		elem := v.Elem()

		if !elem.IsValid() {
			return false
		}

		if elem.Kind() == reflect.Struct {
			return true
		}
	}

	return false
}

func isMarkdownTagEnabled(tag reflect.StructTag) bool {
	tagval := tag.Get(structTagKey)
	switch strings.ToLower(tagval) {
	case "on", "yes", "1", "y", "enable":
		return true
	}
	return false
}

func isStructFieldTagEnabled(structval reflect.Value, fieldIdx int) bool {
	if structval.Kind() != reflect.Struct {
		return false
	}

	structfield := structval.Type().Field(fieldIdx)
	fieldtag := structfield.Tag
	return isMarkdownTagEnabled(fieldtag)
}

func isValidSettable(v reflect.Value) bool {
	return v.IsValid() && v.CanSet()
}

func makeFieldProcessor(c *converter, opts ...parser.ParseOption) *fieldProcessor {
	return &fieldProcessor{
		converter:    c,
		parseOptions: opts,
	}
}
