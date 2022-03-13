package markstruct

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type MyStruct struct {
	Comment string
}

type MyAnnotatedEnabledStruct struct {
	Comment string `markdown:"on"`
}

type MyAnnotatedDisabledStruct struct {
	Comment string `markdown:"off"`
}

type ExplodingMarkdown struct {
	goldmark.Markdown
}

var _ goldmark.Markdown = (*ExplodingMarkdown)(nil)

func (e *ExplodingMarkdown) Convert(_ []byte, _ io.Writer, _ ...parser.ParseOption) error {
	return errors.New("BOOM")
}

func isInvalidType(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrInvalidType)
}

func TestConvertNil(t *testing.T) {
	changed, err := ConvertFields(nil)
	assert.False(t, changed)
	assert.Nil(t, err)

	changed, err = ConvertAllFields(nil)
	assert.False(t, changed)
	assert.Nil(t, err)

	changed, err = ConvertFields((*struct{})(nil))
	assert.False(t, changed)
	assert.Nil(t, err)

	changed, err = ConvertAllFields((*struct{})(nil))
	assert.False(t, changed)
	assert.Nil(t, err)
}

func TestConvertInvalidTypes(t *testing.T) {
	var mystr string

	changed, err := ConvertFields(mystr)
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Equal(t, "", mystr)

	changed, err = ConvertFields(&mystr)
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Equal(t, "", mystr)

	var mylist []string

	changed, err = ConvertFields(mylist)
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Nil(t, mylist)

	changed, err = ConvertFields(&mylist)
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Nil(t, mylist)

	mylist = []string{}

	changed, err = ConvertFields(mylist)
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Equal(t, []string{}, mylist)

	changed, err = ConvertFields(&mylist)
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Equal(t, []string{}, mylist)

	mystruct := MyStruct{}

	changed, err = ConvertFields(mystruct) // should be pointer
	assert.False(t, changed)
	assert.True(t, isInvalidType(err))
	assert.Equal(t, MyStruct{}, mystruct)
}

func TestConvertStringFields(t *testing.T) {
	enabled := &MyAnnotatedEnabledStruct{
		Comment: "_mine_",
	}

	disabled := &MyAnnotatedDisabledStruct{
		Comment: "_mine_",
	}

	c, err := ConvertFields(enabled)
	assert.True(t, c)
	assert.Nil(t, err)

	c, err = ConvertFields(disabled)
	assert.False(t, c)
	assert.Nil(t, err)

	assert.Contains(t, enabled.Comment, "<em>mine</em>") // emphasis mine
	assert.Contains(t, disabled.Comment, "_mine_")
}

func TestConvertAllStringFields(t *testing.T) {
	enabled := &MyAnnotatedEnabledStruct{
		Comment: "_mine_",
	}

	disabled := &MyAnnotatedDisabledStruct{
		Comment: "_mine_",
	}

	c, err := ConvertAllFields(enabled)
	assert.True(t, c)
	assert.Nil(t, err)

	c, err = ConvertAllFields(disabled)
	assert.True(t, c)
	assert.Nil(t, err)

	assert.Equal(t, "<p><em>mine</em></p>\n", enabled.Comment)
	assert.Equal(t, "<p><em>mine</em></p>\n", disabled.Comment)
}

func TestConvertStringFieldsNoMarkdown(t *testing.T) {
	enabled := &MyAnnotatedEnabledStruct{
		Comment: "nothing",
	}

	disabled := &MyAnnotatedDisabledStruct{
		Comment: "nothing",
	}

	c, err := ConvertFields(enabled)
	assert.True(t, c)
	assert.Nil(t, err)

	c, err = ConvertFields(disabled)
	assert.False(t, c)
	assert.Nil(t, err)

	assert.Equal(t, "<p>nothing</p>\n", enabled.Comment)
	assert.Equal(t, "nothing", disabled.Comment)
}

func TestConvertPtrToString(t *testing.T) {
	type TestStruct struct {
		Desc    string  `markdown:"on"`
		DestPtr *string `markdown:"on"`
		Notes   *string
	}

	c, err := ConvertFields(&TestStruct{})
	assert.False(t, c)
	assert.NoError(t, err)

	desc := "_mine_"
	notes := "boop"

	test := &TestStruct{
		Desc:    desc,
		DestPtr: &desc,
		Notes:   &notes,
	}

	c, err = ConvertFields(test)
	assert.True(t, c)
	assert.NoError(t, err)

	assert.Equal(t, "<p><em>mine</em></p>\n", test.Desc)
	assert.NotNil(t, test.DestPtr)
	assert.Equal(t, "<p><em>mine</em></p>\n", *test.DestPtr)
	assert.Equal(t, notes, *test.Notes)
}

func TestConvertStringSlice(t *testing.T) {
	type Test struct {
		Marked      []string `markdown:"on"`
		Unmarked    []string
		MarkedPtr   *[]string `markdown:"on"`
		UnmarkedPtr *[]string
	}

	d1 := []string{"_one_", "_two_", "_three_"}
	d2 := []string{"_one_", "_two_", "_three_"}

	test := &Test{
		Marked:      []string{"_one_", "_two_", "_three_"},
		Unmarked:    []string{"_one_", "_two_", "_three_"},
		MarkedPtr:   &d1,
		UnmarkedPtr: &d2,
	}

	c, err := ConvertFields(test)
	assert.True(t, c)
	assert.NoError(t, err)

	assert.Equal(
		t,
		[]string{
			"<p><em>one</em></p>\n",
			"<p><em>two</em></p>\n",
			"<p><em>three</em></p>\n",
		},
		test.Marked,
	)

	assert.Equal(
		t,
		[]string{"_one_", "_two_", "_three_"},
		test.Unmarked,
	)

	assert.Equal(
		t,
		[]string{
			"<p><em>one</em></p>\n",
			"<p><em>two</em></p>\n",
			"<p><em>three</em></p>\n",
		},
		d1,
	)

	assert.Equal(
		t,
		[]string{"_one_", "_two_", "_three_"},
		d2,
	)
}

func TestConvertNonStringList(t *testing.T) {
	type Test struct {
		Scores []int `markdown:"on"`
		Deltas []int
	}

	test := &Test{
		Scores: []int{80, 90, 100},
		Deltas: []int{20, 10, 0},
	}

	c, err := ConvertFields(test)
	assert.False(t, c)
	assert.NoError(t, err)

	assert.Equal(t, []int{80, 90, 100}, test.Scores)
	assert.Equal(t, []int{20, 10, 0}, test.Deltas)
}

func TestConvertNilSlice(t *testing.T) {
	type Test struct {
		Marked      []string `markdown:"on"`
		Unmarked    []string
		MarkedPtr   *[]string `markdown:"on"`
		UnmarkedPtr *[]string
	}

	test := &Test{
		Marked:      nil,
		Unmarked:    nil,
		MarkedPtr:   (*[]string)(nil),
		UnmarkedPtr: (*[]string)(nil),
	}

	c, err := ConvertFields(test)
	assert.False(t, c)
	assert.NoError(t, err)

	assert.Nil(t, test.Marked)
	assert.Nil(t, test.Unmarked)
	assert.Nil(t, test.MarkedPtr)
	assert.Nil(t, test.UnmarkedPtr)
}

func TestConvertMap(t *testing.T) {
	type Test struct {
		Codes          map[int]string
		FormattedCodes map[int]string `markdown:"on"`
	}

	test := &Test{
		Codes: map[int]string{
			200: "**OK**",
			500: "_Error_",
		},

		FormattedCodes: map[int]string{
			200: "**OK**",
			500: "_Error_",
		},
	}

	c, err := ConvertFields(test)
	assert.True(t, c)
	assert.NoError(t, err)

	assert.Equal(
		t,
		"**OK**",
		test.Codes[200],
	)

	assert.Equal(
		t,
		"_Error_",
		test.Codes[500],
	)

	assert.Equal(
		t,
		"<p><strong>OK</strong></p>\n",
		test.FormattedCodes[200],
	)

	assert.Equal(
		t,
		"<p><em>Error</em></p>\n",
		test.FormattedCodes[500],
	)
}

func TestConvertNilMap(t *testing.T) {
	type Test struct {
		Codes          map[int]string
		FormattedCodes map[int]string `markdown:"on"`
	}

	test := &Test{
		Codes:          nil,
		FormattedCodes: nil,
	}

	c, err := ConvertFields(test)
	assert.False(t, c)
	assert.NoError(t, err)

	assert.Nil(t, test.Codes)
	assert.Nil(t, test.FormattedCodes)
}

func TestConvertNestedStruct(t *testing.T) {
	type PersonalDetails struct {
		FullName    string
		Description string `markdown:"on"`
	}

	type Employee struct {
		ID      int
		Details PersonalDetails
		Manager *Employee
	}

	employee42 := &Employee{
		ID: 42,
		Details: PersonalDetails{
			FullName:    "Al Choholic",
			Description: "Part of the _Sales_ Team",
		},
		Manager: &Employee{
			ID: 10,
			Details: PersonalDetails{
				FullName:    "Gordon Gecko",
				Description: "_Sales_ Team **Lead**",
			},
		},
	}

	c, err := ConvertFields(employee42)
	assert.True(t, c)
	assert.NoError(t, err)

	assert.Equal(t, "Al Choholic", employee42.Details.FullName)
	assert.Equal(t, "Gordon Gecko", employee42.Manager.Details.FullName)

	assert.Equal(
		t,
		"<p>Part of the <em>Sales</em> Team</p>\n",
		employee42.Details.Description,
	)

	assert.Equal(
		t,
		"<p><em>Sales</em> Team <strong>Lead</strong></p>\n",
		employee42.Manager.Details.Description,
	)
}

func TestConvertNestedNilStructs(t *testing.T) {
	type PersonalDetails struct {
		FullName    string
		Description string `markdown:"on"`
	}

	type Employee struct {
		ID      int
		Details PersonalDetails
		Manager *Employee
	}

	empnomanager := &Employee{
		Manager: (*Employee)(nil),
	}

	c, err := ConvertFields(empnomanager)
	assert.False(t, c)
	assert.NoError(t, err)
	assert.Nil(t, empnomanager.Manager)

	empnomanager = (*Employee)(nil)

	c, err = ConvertFields(empnomanager)
	assert.False(t, c)
	assert.NoError(t, err)
	assert.Nil(t, empnomanager)
}

func TestIsValidSettable(t *testing.T) {
	myfoo := &MyStruct{}

	value := reflect.ValueOf(myfoo)
	assert.False(t, isValidSettable(value))

	elem := value.Elem()
	assert.True(t, isValidSettable(elem))
}

func TestMarkdownTagEnabled(t *testing.T) {
	absent := reflect.ValueOf(MyStruct{})
	disabled := reflect.ValueOf(MyAnnotatedDisabledStruct{})
	enabled := reflect.ValueOf(MyAnnotatedEnabledStruct{})

	absent0 := absent.Type().Field(0)
	disabled0 := disabled.Type().Field(0)
	enabled0 := enabled.Type().Field(0)

	assert.False(t, isMarkdownTagEnabled(absent0.Tag))
	assert.False(t, isMarkdownTagEnabled(disabled0.Tag))
	assert.True(t, isMarkdownTagEnabled(enabled0.Tag))
}

func TestWithMarkdown(t *testing.T) {
	teststr := "~~strike~~"
	testDefault := &MyAnnotatedEnabledStruct{teststr}
	testCustom := &MyAnnotatedEnabledStruct{teststr}

	customMd := WithMarkdown(
		goldmark.New(
			goldmark.WithExtensions(
				extension.Strikethrough,
			),
		),
	)

	c, err := ConvertFields(testDefault)
	assert.True(t, c)
	assert.NoError(t, err)

	c, err = customMd.ConvertFields(testCustom)
	assert.True(t, c)
	assert.NoError(t, err)

	assert.Equal(t, "<p>~~strike~~</p>\n", testDefault.Comment)
	assert.Equal(t, "<p><del>strike</del></p>\n", testCustom.Comment)
}

func TestValidateFields(t *testing.T) {
	plain := "Hello *World*"
	converted := "<p>Hello <em>World</em></p>\n"

	object1 := MyAnnotatedEnabledStruct{
		Comment: plain,
	}

	object2 := object1

	changed, err := ConvertFields(&object1)
	assert.True(t, changed, err)
	assert.NoError(t, err)

	assert.Equal(t, converted, object1.Comment)

	changed, err = ValidateFields(&object2)
	assert.True(t, changed)
	assert.NoError(t, err)

	assert.Equal(t, plain, object2.Comment)
}

func TestValidateAllFields(t *testing.T) {
	plain := "Hello *World*"
	converted := "<p>Hello <em>World</em></p>\n"

	object1 := MyAnnotatedDisabledStruct{
		Comment: plain,
	}

	object2 := object1

	changed, err := ConvertAllFields(&object1)
	assert.True(t, changed, err)
	assert.NoError(t, err)

	assert.Equal(t, converted, object1.Comment)

	changed, err = ValidateAllFields(&object2)
	assert.True(t, changed)
	assert.NoError(t, err)

	assert.Equal(t, plain, object2.Comment)
}

func TestConvertFieldsWithError(t *testing.T) {
	badconverter := WithMarkdown(&ExplodingMarkdown{})

	myobj := &MyAnnotatedEnabledStruct{
		Comment: "Hello World",
	}

	changed, err := badconverter.ConvertFields(myobj)
	assert.False(t, changed)
	assert.Error(t, err)

	otherobj := &MyAnnotatedDisabledStruct{
		Comment: "Hello World",
	}

	changed, err = badconverter.ConvertFields(otherobj)
	assert.False(t, changed)
	assert.NoError(t, err)
}

func TestConvertAllFieldsWithError(t *testing.T) {
	badconverter := WithMarkdown(&ExplodingMarkdown{})

	myobj := &MyAnnotatedDisabledStruct{
		Comment: "Hello World",
	}

	changed, err := badconverter.ConvertAllFields(myobj)
	assert.False(t, changed)
	assert.Error(t, err)

	otherobj := &MyAnnotatedDisabledStruct{
		Comment: "Hello World",
	}

	changed, err = badconverter.ConvertAllFields(otherobj)
	assert.False(t, changed)
	assert.Error(t, err)
}

func TestValidateFieldsWithError(t *testing.T) {
	badconverter := WithMarkdown(&ExplodingMarkdown{})

	myobj := &MyAnnotatedEnabledStruct{
		Comment: "Hello World",
	}

	changed, err := badconverter.ValidateFields(myobj)
	assert.False(t, changed)
	assert.Error(t, err)

	otherobj := &MyAnnotatedDisabledStruct{
		Comment: "Hello World",
	}

	changed, err = badconverter.ValidateFields(otherobj)
	assert.False(t, changed)
	assert.NoError(t, err)
}

func TestValidateAllFieldsWithError(t *testing.T) {
	badconverter := WithMarkdown(&ExplodingMarkdown{})

	myobj := &MyAnnotatedEnabledStruct{
		Comment: "Hello World",
	}

	changed, err := badconverter.ValidateAllFields(myobj)
	assert.False(t, changed)
	assert.Error(t, err)

	otherobj := &MyAnnotatedDisabledStruct{
		Comment: "Hello World",
	}

	changed, err = badconverter.ValidateAllFields(otherobj)
	assert.False(t, changed)
	assert.Error(t, err)
}

func TestConvertWronglyTypedFields(t *testing.T) {
	type Something struct {
		Name      string
		Details   string
		Activated bool `markdown:"on"` // wrong type, not a string
		Level     int  `markdown:"on"` // wrong type, not a string
	}

	alpha1 := &Something{
		Name:      "alpha",
		Details:   "Highly *developed*",
		Activated: true,
		Level:     1,
	}

	changed, err := ConvertFields(alpha1)
	assert.False(t, changed)
	assert.NoError(t, err)

	assert.Equal(t, "alpha", alpha1.Name)
	assert.Equal(t, "Highly *developed*", alpha1.Details)
	assert.True(t, alpha1.Activated)
	assert.Equal(t, 1, alpha1.Level)

	changed, err = ConvertAllFields(alpha1)
	assert.True(t, changed)
	assert.NoError(t, err)

	assert.Equal(t, "<p>alpha</p>\n", alpha1.Name)
	assert.Equal(t, "<p>Highly <em>developed</em></p>\n", alpha1.Details)
	assert.True(t, alpha1.Activated)
	assert.Equal(t, 1, alpha1.Level)
}

func TestConvertMapValue(t *testing.T) {
	fieldproc := makeFieldProcessor(defaultConverter.(*converter))

	foo := "Hello World"

	test := map[string]string{
		"description": "Hello *World*!",
	}

	changed, err := fieldproc.convertMap(reflect.ValueOf(nil))
	assert.False(t, changed)
	assert.Error(t, err)
	assert.True(t, isInvalidType(err))

	changed, err = fieldproc.convertMap(reflect.ValueOf(foo))
	assert.False(t, changed)
	assert.Error(t, err)
	assert.True(t, isInvalidType(err))

	changed, err = fieldproc.convertMap(reflect.ValueOf([]string{}))
	assert.False(t, changed)
	assert.Error(t, err)
	assert.True(t, isInvalidType(err))

	changed, err = fieldproc.convertMap(reflect.ValueOf(test))
	assert.False(t, changed) // false as direct map value is not settable
	assert.NoError(t, err)
	assert.Equal(t, "Hello *World*!", test["description"])
}

func TestConvertSliceValue(t *testing.T) {
	fieldproc := makeFieldProcessor(defaultConverter.(*converter))

	foo := "Hello World"

	test := []string{"one", "two", "three"}

	changed, err := fieldproc.convertSlice(reflect.ValueOf(nil))
	assert.False(t, changed)
	assert.Error(t, err)
	assert.True(t, isInvalidType(err))

	changed, err = fieldproc.convertSlice(reflect.ValueOf(foo))
	assert.False(t, changed)
	assert.Error(t, err)
	assert.True(t, isInvalidType(err))

	changed, err = fieldproc.convertSlice(reflect.ValueOf(map[string]string{}))
	assert.False(t, changed)
	assert.Error(t, err)
	assert.True(t, isInvalidType(err))

	changed, err = fieldproc.convertSlice(reflect.ValueOf(test))
	assert.True(t, changed)
	assert.NoError(t, err)
	assert.Equal(t, "<p>one</p>\n", test[0])
}
