package typex

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

type LogData struct {
	Path     string
	Query    string
	QueryMap map[string]any

	Name    string
	Method  string
	ReqBody string
	ResBody string
}

func ParseLogData(fp string) (*LogData, error) {
	var rtn = LogData{QueryMap: make(map[string]any)}
	f, err := os.OpenFile(fp, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	strData := string(data)
	strArr := strings.Split(strData, "\n")

	for _, str := range strArr {
		hd := strings.SplitN(str, ":", 2)
		if len(hd) != 2 {
			continue
		}
		head := hd[0]
		cnt := hd[1]
		switch strings.TrimSpace(head) {
		case "path":
			rtn.Path = strings.TrimSpace(cnt)
		case "query":
			rtn.Query = strings.TrimSpace(cnt)
		case "method":
			rtn.Method = strings.ToLower(strings.TrimSpace(cnt))
		case "name":
			rtn.Name = strings.TrimSpace(cnt)
		case "reqbody":
			rtn.ReqBody = strings.TrimSpace(cnt)
		case "resbody":
			rtn.ResBody = strings.TrimSpace(cnt)
		}
	}

	return &rtn, nil
}

func ParseLog(fp string) ([]string, error) {
	var rtn []string

	var rtnData, err = ParseLogData(fp)
	if err != nil {
		return nil, err
	}

	var (
		queryName string
		reqName   string
		resName   string

		upperName = strcase.ToCamel(rtnData.Name)
	)

	if rtnData.Query != "" {
		queryName = strcase.ToCamel(rtnData.Method + fmt.Sprintf("%sQuery", upperName))
		qJ, err := query2JsonStr(rtnData.Query)
		if err != nil {
			return nil, err
		}
		qTs, err := jsonToTypeScriptInterface(qJ, queryName)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, qTs)
	}

	if rtnData.ReqBody != "" {
		reqName = strcase.ToCamel(rtnData.Method + fmt.Sprintf("%sReq", upperName))
		reqTs, err := jsonToTypeScriptInterface(rtnData.ReqBody, reqName)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, reqTs)
	}

	if rtnData.ResBody != "" {
		resName = strcase.ToCamel(rtnData.Method + fmt.Sprintf("%sRes", upperName))
		resTs, err := jsonToTypeScriptInterface(rtnData.ResBody, resName)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, resTs)
	}

	mtd := generateMethod(rtnData.Path, rtnData.Name, rtnData.Method, reqName, queryName, resName)
	if mtd != "" {
		rtn = append(rtn, mtd)
	}

	return rtn, nil
}

func generateMethod(path, name, method, reqName, queryName, resName string) string {
	var strTemplate = `const {{.MethodName}} = ({{.ReqParams}}) => {
  return request<{{.ResName}}>({
    url: '{{.Path}}',
    method: '{{.Method}}',
	{{if .ReqName -}}	data: payload, {{- end -}}
	{{if .QueryName -}} params: query, {{- end}}
  });
};`

	var reqParamsBuilder = strings.Builder{}
	if queryName != "" {
		reqParamsBuilder.WriteString("query: " + queryName)
		reqParamsBuilder.WriteString(", ")
	}
	if reqName != "" {
		reqParamsBuilder.WriteString("payload: " + reqName)
		reqParamsBuilder.WriteString(", ")
	}

	var reqParams = strings.TrimRight(reqParamsBuilder.String(), ", ")

	var data = map[string]any{
		"Path":       path,
		"Method":     method,
		"ReqName":    reqName,
		"ReqParams":  reqParams,
		"MethodName": fmt.Sprintf("%sApi%s", strings.ToLower(method), strcase.ToCamel(name)),
		"ResName":    resName,
		"QueryName":  queryName,
	}

	var res bytes.Buffer

	temp, err := template.New("ts-mt").Parse(strTemplate)
	if err != nil {
		return ""
	}

	err = temp.Execute(&res, data)
	if err != nil {
		return ""
	}

	return res.String()
}
