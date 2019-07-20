package module

import (
	"bytes"
	"html/template"
	"strings"
)

type UIBuilder struct {
	builder strings.Builder
}

var inputTextTempl *template.Template

func init() {
	var err error
	inputTextTempl, err = template.New("input").Parse(
		`    <input type="text" id="{{.Id}}" name="{{.Name}}" value="{{.Value}}"/>
`)
	if err != nil {
		panic(err)
	}
}

func (b *UIBuilder) StartForm(module Module, relativePath string, legend string) *UIBuilder {
	b.builder.WriteString(`<form class="pure-form" action="`)
	b.builder.WriteString("/" + module.InternalName() + "/" + relativePath)
	b.builder.WriteString(`" method="post" accept-charset="UTF-8">
  <fieldset>
`)
	if legend != "" {
		b.builder.WriteString(`    <legend>` + legend + `</legend>
`)
	}
	b.builder.WriteString(`    <input type="hidden" name="redirect" value="1"/>`)
	return b
}

func (b *UIBuilder) EndForm() *UIBuilder {
	b.builder.WriteString(`  </fieldset>
</form>
`)
	return b
}

func (b *UIBuilder) StartSelect(label string, id string, name string) *UIBuilder {
	b.builder.WriteString(`    <label for="`)
	b.builder.WriteString(id)
	b.builder.WriteString(`">`)
	b.builder.WriteString(label)
	b.builder.WriteString(`</label>
    <select id="`)
	b.builder.WriteString(id)
	b.builder.WriteString(`" name="`)
	b.builder.WriteString(name)
	b.builder.WriteString(`">
`)
	return b
}

func (b *UIBuilder) Option(value string, selected bool, content string) *UIBuilder {
	b.builder.WriteString(`      <option value="`)
	b.builder.WriteString(value)
	if selected {
		b.builder.WriteString(`" selected="selected">`)
	} else {
		b.builder.WriteString(`">`)
	}
	b.builder.WriteString(content)
	b.builder.WriteString(`</option>
`)
	return b
}

func (b *UIBuilder) EndSelect() *UIBuilder {
	b.builder.WriteString(`    </select>
`)
	return b
}

func (b *UIBuilder) TextInput(label string, id string, name string, value string) *UIBuilder {
	var buf bytes.Buffer
	if err := inputTextTempl.Execute(&buf, struct {
		Id    string
		Name  string
		Value string
	}{
		Id: id, Name: name, Value: value}); err != nil {
		panic(err)
	}
	b.builder.WriteString(`    <label for="`)
	b.builder.WriteString(id)
	b.builder.WriteString(`">`)
	b.builder.WriteString(label)
	b.builder.WriteString(`</label>
`)
	b.builder.WriteString(buf.String())
	return b
}

func (b *UIBuilder) SubmitButton(caption string) *UIBuilder {
	b.builder.WriteString(`    <button type="submit" class="pure-button pure-button-primary">`)
	b.builder.WriteString(caption)
	b.builder.WriteString(`</button>
`)
	return b
}

func (b *UIBuilder) Finish() template.HTML {
	return template.HTML(b.builder.String())
}
