# markstruct

> markstruct converts a struct's string fields from Markdown to HTML in-place.

`markstruct` scans a struct for tagged fields of relevant type (`string`, `*string`, `[]string` & maps with `string` values), and renders the field value from Markdown to HTML in-place. That is to say the value of each field itself will be changed within the struct to be the HTML result of rendering the original value as Markdown.

`markstruct` uses `github.com/yuin/goldmark` to render Markdown, and allows for
 custom `goldmark.Markdown` objects and parse options.

 ## Installation

 ```bash
 go get github.com/herbygillot/markstruct
 ```

 ## Usage

 `ConvertFields` accepts a pointer to struct, and converts fields of relevant type (as listed above) from Markdown to HTML that are tagged with `markdown:"on"`:
 ```

 type Document struct {
   Title string                 // this field will be ignored
   Body  string `markdown:"on"` // this field will be converted
 }

  doc := &Document{
	  Title: "Doc *1*",
	  Body:  "This is _emphasis_.",
  }

  changed, err := markstruct.ConvertFields(doc)
  ...
  fmt.Println(doc.Title) // "Doc *1*"
  fmt.Println(doc.Body)  // "<p>This is <em>emphasis</em>.</p>"

 ```

 `ConvertAllFields` also accepts a pointer to struct, but will convert **all** fields of relevant type, ignoring the absence or presence of the `markdown:"on"` tag.

 ```
 type Document struct {
   Title string                 // normally ignored, but will be converted by ConvertAllFields
   Body  string `markdown:"on"` // this field will be converted
 }

  doc := &Document{
	  Title: "Doc *1*",
	  Body:  "This is _emphasis_.",
  }

  changed, err := markstruct.ConvertAllFields(doc)
  ...
  fmt.Println(doc.Title) // "<p>Doc <em>1</em></p>"
  fmt.Println(doc.Body)  // "<p>This is <em>emphasis</em>.</p>"
  ```