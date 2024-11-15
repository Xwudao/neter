{{- /*gotype: github.com/Xwudao/neter/cmd/nr/cmd.Generator*/ -}}

package params

import (
	"{{.ModName}}/internal/routes/valid"
)

type Create{{.EntName}}Params struct {
    // todo: add fields
}

func (c *Create{{.EntName}}Params) GetMessages() valid.ValidatorMessages {
	return valid.ValidatorMessages{
	}
}

type Update{{.EntName}}Params struct {
	ID int64 `json:"id" binding:"required"`
    // add fields
}

func (u *Update{{.EntName}}Params) GetMessages() valid.ValidatorMessages {
	return valid.ValidatorMessages{
		"ID.required": "ID必填",
	}
}

type List{{.EntName}}Params struct {
	Page int `json:"page" form:"page" binding:"min=1"`
	Size int `json:"size" form:"size" binding:"min=1,max=50"`

	ByID string `json:"by_id" form:"by_id" binding:"oneof=asc desc"`

	Offset int `json:"-" form:"-"`
}

func (l *List{{.EntName}}Params) Optimize() error {
	if l.Page == 0 {
		l.Page = 1
	}
	if l.Size == 0 {
		l.Size = 10
	}
	l.Offset = (l.Page - 1) * l.Size
	return nil
}

func (l *List{{.EntName}}Params) GetMessages() valid.ValidatorMessages {
	return valid.ValidatorMessages{
		"Page.min":   "页码不能小于1",
		"Size.min":   "每页数量不能小于1",
		"Size.max":   "每页数量不能大于50",
		"ByID.oneof": "排序方式只能是 asc 或 desc",
	}
}

type Count{{.EntName}}Params struct {
    // todo: add fields
}
