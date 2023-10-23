package typex

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Xwudao/neter/pkg/json2go"
	"github.com/iancoleman/strcase"
)

var goTemplate = `package client

import (
	"time"

	"github.com/imroc/req/v3"
)

type ApiClient struct {
	r *req.Client
}

func NewApiClient() *ApiClient {
	var r = req.NewClient().SetTimeout(10 * time.Second)
	return &ApiClient{r: r}
}

func (c *ApiClient) {{.MethodName}}({{if .HasReq}}data *{{.ReqName}} {{end -}}) (*{{.RespName}}, error) {
	var respData = new({{.RespName}})

	var builder = c.r.R()
	{{- if .HasReq}}
	builder.SetBodyJsonMarshal(data)
	{{- end}}
	{{- if .HasQuery}}
	{{- range $key,$value := .QueryMap}}
		builder.SetQueryParam("{{$key}}", "{{$value}}")
	{{- end}}
	{{- end}}
	builder.SetSuccessResult(respData)

	if _, err := builder.{{.ReqMethod}}("{{.Path}}"); err != nil {
		return nil, err
	}
	return respData, nil
}`

func Parse2Go(fp string) ([]string, error) {
	var rtnStr []string

	var rtnData, err = ParseLogData(fp)
	if err != nil {
		return nil, err
	}

	var mapData = map[string]any{
		"MethodName": fmt.Sprintf(`%sApi%s`, strcase.ToCamel(rtnData.Method), strcase.ToCamel(rtnData.Name)),
		"ReqMethod":  strcase.ToCamel(rtnData.Method),
		"HasReq":     rtnData.ReqBody != "",
		"HasQuery":   len(rtnData.QueryMap) > 0,
		"ReqName":    fmt.Sprintf(`%sApi%sReq`, strcase.ToCamel(rtnData.Method), strcase.ToCamel(rtnData.Name)),
		"RespName":   fmt.Sprintf(`%sApi%sResp`, strcase.ToCamel(rtnData.Method), strcase.ToCamel(rtnData.Name)),
		"Path":       rtnData.Path,
		"QueryMap":   rtnData.QueryMap,
	}

	var tpl = template.New("gen-go")
	tpl, err = tpl.Parse(goTemplate)
	if err != nil {
		return nil, err
	}

	var buf = new(bytes.Buffer)
	if err := tpl.Execute(buf, mapData); err != nil {
		return nil, err
	}

	rtnStr = append(rtnStr, buf.String())

	if rtnData.ReqBody != "" {
		json2Go := json2go.NewJson2Go(mapData["ReqName"].(string))
		reqResult, err := json2Go.Generate(rtnData.ReqBody)
		if err == nil {
			rtnStr = append(rtnStr, reqResult)
		}
	}

	if rtnData.ResBody != "" {
		json2Go := json2go.NewJson2Go(mapData["RespName"].(string))
		resResult, err := json2Go.Generate(rtnData.ResBody)
		if err == nil {
			rtnStr = append(rtnStr, resResult)
		}
	}

	return rtnStr, nil
}
