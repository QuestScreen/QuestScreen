package display

import (
	"bytes"
	"html/template"
	"strings"
)

type UIBuilder struct {
	builder     strings.Builder
	formAligned bool
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

func (b *UIBuilder) StartGroup() *UIBuilder {
	b.builder.WriteString(`    <div class="pure-control-group">
`)
	return b
}

func (b *UIBuilder) EndGroup() *UIBuilder {
	b.builder.WriteString(`    </div>
`)
	return b
}

func (b *UIBuilder) StartForm(module Module, relativePath string, legend string, aligned bool) *UIBuilder {
	b.builder.WriteString(`<form class="pure-form`)
	if aligned {
		b.builder.WriteString(` pure-form-aligned`)
	}
	b.builder.WriteString(`" action="`)
	b.builder.WriteString("/" + module.InternalName() + "/" + relativePath)
	b.builder.WriteString(`" method="post" accept-charset="UTF-8">
  <fieldset>
`)
	if legend != "" {
		b.builder.WriteString(`    <legend>` + legend + `</legend>
`)
	}
	b.builder.WriteString(`    <input type="hidden" name="redirect" value="1"/>`)
	b.formAligned = aligned
	return b
}

func (b *UIBuilder) EndForm() *UIBuilder {
	b.builder.WriteString(`  </fieldset>
</form>
`)
	return b
}

func (b *UIBuilder) StartSelect(label string, id string, name string) *UIBuilder {
	if b.formAligned {
		b.StartGroup()
	}
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
	if b.formAligned {
		b.EndGroup()
	}
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
	if b.formAligned {
		b.StartGroup()
	}
	b.builder.WriteString(`    <label for="`)
	b.builder.WriteString(id)
	b.builder.WriteString(`">`)
	b.builder.WriteString(label)
	b.builder.WriteString(`</label>
`)
	b.builder.WriteString(buf.String())
	if b.formAligned {
		b.EndGroup()
	}
	return b
}

func (b *UIBuilder) SubmitButton(caption string, label string, enabled bool) *UIBuilder {
	if b.formAligned {
		b.StartGroup()
	}
	if label != "" {
		b.builder.WriteString(`    <label>`)
		b.builder.WriteString(label)
		b.builder.WriteString(`</label>
`)
	}
	b.builder.WriteString(`    <button type="submit" class="pure-button pure-button-primary`)
	if !enabled {
		b.builder.WriteString(" pure-button-disabled")
	}
	b.builder.WriteString(`">`)
	b.builder.WriteString(caption)
	b.builder.WriteString(`</button>
`)
	if b.formAligned {
		b.EndGroup()
	}
	return b
}

func (b *UIBuilder) SecondarySubmitButton(caption string, label string, enabled bool) *UIBuilder {
	if b.formAligned {
		b.StartGroup()
	}
	if label != "" {
		b.builder.WriteString(`    <label>`)
		b.builder.WriteString(label)
		b.builder.WriteString(`</label>
`)
	}
	b.builder.WriteString(`    <button type="submit" class="pure-button`)
	if !enabled {
		b.builder.WriteString(" pure-button-disabled")
	}
	b.builder.WriteString(`">`)
	b.builder.WriteString(caption)
	b.builder.WriteString(`</button>
`)
	if b.formAligned {
		b.EndGroup()
	}
	return b
}

func (b *UIBuilder) HiddenValue(name string, value string) *UIBuilder {
	b.builder.WriteString(`    <input type="hidden" name="`)
	b.builder.WriteString(name)
	b.builder.WriteString(`" value="`)
	b.builder.WriteString(value)
	b.builder.WriteString(`"/>
`)
	return b
}

func (b *UIBuilder) Finish() template.HTML {
	return template.HTML(b.builder.String())
}
