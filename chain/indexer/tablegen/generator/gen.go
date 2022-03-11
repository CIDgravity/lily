package generator

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"text/template"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/storage"
)

func Gen() error {
	taskDir := "./chain/indexer"
	rf, err := ioutil.ReadFile(filepath.Join(taskDir, "table_tasks.go.template"))
	if err != nil {
		return xerrors.Errorf("loading registry template: %w", err)
	}

	mtn := modelTableNames()
	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"tableNames": func(v int) ModelTypeNames { return mtn[v] },
	}).Parse(string(rf)))

	var b bytes.Buffer
	if err := tpl.Execute(&b, map[string]interface{}{
		"tableNames": mtn,
	}); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(taskDir, "table_tasks.go"), b.Bytes(), 0o666); err != nil {
		return err
	}
	return nil
}

func modelTableNames() []ModelTypeNames {
	var out []ModelTypeNames
	for _, model := range storage.Models {
		name := getModelTableName(reflect.TypeOf(model).Elem())
		out = append(out, name)
	}
	return out
}

type ModelTypeNames struct {
	TypeName  string
	ModelName string
}

func getModelTableName(t reflect.Type) ModelTypeNames {
	typeName := ToExported(t.Name())
	modelName := Underscore(t.Name())
	// if the struct is tagged with a pg table name tag use that instead
	if f, has := t.FieldByName("tableName"); has {
		modelName = f.Tag.Get("pg")
	}
	return ModelTypeNames{
		TypeName:  typeName,
		ModelName: modelName,
	}
}

func ToExported(s string) string {
	if len(s) == 0 {
		return s
	}
	if c := s[0]; IsLower(c) {
		b := []byte(s)
		b[0] = ToUpper(c)
		return string(b)
	}
	return s
}

func IsLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func ToUpper(c byte) byte {
	return c - 32
}

// Underscore converts "CamelCasedString" to "camel_cased_string".
func Underscore(s string) string {
	r := make([]byte, 0, len(s)+5)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if IsUpper(c) {
			if i > 0 && i+1 < len(s) && (IsLower(s[i-1]) || IsLower(s[i+1])) {
				r = append(r, '_', ToLower(c))
			} else {
				r = append(r, ToLower(c))
			}
		} else {
			r = append(r, c)
		}
	}
	return string(r)
}

func IsUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func ToLower(c byte) byte {
	return c + 32
}
