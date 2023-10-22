package def

import (
	"fmt"
	"go/format"
	"slices"
	"strconv"
	"strings"

	gen "github.com/dave/jennifer/jen"
	"github.com/samber/lo"

	"github.com/Xwudao/neter/pkg/json2go/conv"
)

type Type interface {
	fmt.Stringer
	Code() gen.Code
	Nullable() Type
}

type TypeDecl struct {
	Name string
	Type Type
}

func (t TypeDecl) String() string {
	s := fmt.Sprintf("type %s %s", t.Name, t.Type)
	r, err := format.Source([]byte(s))
	if err != nil {
		return s
	} else {
		return string(r)
	}
}

type Int struct {
	Pointer bool
}

func (i Int) String() string {
	if i.Pointer {
		return "*int"
	} else {
		return "int"
	}
}

func (i Int) Nullable() Type {
	i.Pointer = true
	return i
}

func (i Int) Code() gen.Code {
	if i.Pointer {
		return gen.Op("*").Int()
	}
	return gen.Int()
}

type Float struct {
	Pointer bool
}

func (f Float) String() string {
	if f.Pointer {
		return "*float64"
	} else {
		return "float64"
	}
}

func (f Float) Nullable() Type {
	f.Pointer = true
	return f
}

func (f Float) Code() gen.Code {
	if f.Pointer {
		return gen.Op("*").Float64()
	}
	return gen.Float64()
}

type Bool struct {
	Pointer bool
}

func (b Bool) String() string {
	if b.Pointer {
		return "*bool"
	} else {
		return "bool"
	}
}

func (b Bool) Nullable() Type {
	b.Pointer = true
	return b
}

func (b Bool) Code() gen.Code {
	if b.Pointer {
		return gen.Op("*").Bool()
	}
	return gen.Bool()
}

type String struct {
	Pointer bool
}

func (s String) String() string {
	if s.Pointer {
		return "*string"
	} else {
		return "string"
	}
}

func (s String) Nullable() Type {
	s.Pointer = true
	return s
}

func (s String) Code() gen.Code {
	if s.Pointer {
		return gen.Op("*").String()
	}
	return gen.String()
}

type Array struct {
	Element Type // element type
}

func (a Array) String() string {
	return fmt.Sprintf("[]%s", a.Element)
}

func (a Array) Nullable() Type {
	return a
}

func (a Array) Code() gen.Code {
	return gen.Index().Add(a.Element.Code())
}

type Field struct {
	Name      string // field name
	Key       string // json key
	Type      Type
	OmitEmpty bool
	//String    bool // string option: string, bool, int or float
}

func (f Field) String() string {
	return fmt.Sprintf("%s %s `json:\"%s\"`", f.Name, f.Type, f.Options())
}

func (f Field) Options() string {
	options := make([]string, 0, 3)
	options = append(options, f.Key)

	if f.OmitEmpty {
		options = append(options, "omitempty")
	}
	//if f.String {
	//	options = append(options, "string")
	//}
	return strings.Join(options, ",")
}

func (f Field) Code() gen.Code {
	return gen.Id(f.Name).Add(f.Type.Code(), gen.Tag(map[string]string{"json": f.Options()}))
}

type Struct struct {
	Fields  []*Field
	Pointer bool
	conv.CamelCaseConverter
}

func (s Struct) String() string {
	s.Naming()
	return fmt.Sprintf(
		`struct{
	%s
}`,
		strings.Join(
			lo.Map(s.Fields, func(field *Field, _ int) string {
				return field.String()
			}),
			"\n\t",
		),
	)
}

func (s Struct) Nullable() Type {
	s.Pointer = true
	return s
}

func (s Struct) Naming() {
	if len(s.Fields) == 0 {
		return
	}

	named := !slices.Contains(
		lo.Map(s.Fields, func(field *Field, _ int) string {
			return field.Name
		}),
		"",
	)
	if named {
		return
	}

	if s.Fields[0].Name != "" {
		return
	}

	counter := make(map[string]int)
	for _, field := range s.Fields {
		field.Name = s.ToCamelCase(field.Key)
		if counter[field.Name]++; counter[field.Name] != 1 {
			field.Name += strconv.Itoa(counter[field.Name])
		}
	}
}

func (s Struct) Code() gen.Code {
	s.Naming()
	codes := lo.Map(s.Fields, func(field *Field, _ int) gen.Code {
		return field.Code()
	})

	if s.Pointer {
		gen.Op("*").Struct(codes...)
	}
	return gen.Struct(codes...)
}

type Map struct {
	Key   Type // key type: string or int
	Value Type // value type
}

func (m Map) String() string {
	return fmt.Sprintf("map[%s]%s", m.Key, m.Value)
}

func (m Map) Nullable() Type {
	return m
}

func (m Map) Code() gen.Code {
	return gen.Map(m.Key.Code()).Add(m.Value.Code())
}

type Any struct{}

func (a Any) String() string {
	return "any"
}

func (a Any) Nullable() Type {
	return a
}

func (a Any) Code() gen.Code {
	return gen.Any()
}
