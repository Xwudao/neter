package tsx

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

func ParseLog(fp string) ([]string, error) {
	var rtn []string

	f, err := os.OpenFile(fp, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var (
		path       string
		query      string
		method     string
		name       string
		reqBodyStr string
		resBodyStr string
	)

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
			path = strings.TrimSpace(cnt)
		case "query":
			query = strings.TrimSpace(cnt)
		case "method":
			method = strings.ToLower(strings.TrimSpace(cnt))
		case "name":
			name = strings.TrimSpace(cnt)
		case "reqbody":
			reqBodyStr = strings.TrimSpace(cnt)
		case "resbody":
			resBodyStr = strings.TrimSpace(cnt)
		}
	}

	var (
		queryName string
		reqName   string
		resName   string

		upperName = strcase.ToCamel(name)
	)

	if query != "" {
		queryName = strcase.ToCamel(method + fmt.Sprintf("%sQuery", upperName))
		qJ, err := query2JsonStr(query)
		if err != nil {
			return nil, err
		}
		qTs, err := jsonToTypeScriptInterface(qJ, queryName)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, qTs)
	}

	if reqBodyStr != "" {
		reqName = strcase.ToCamel(method + fmt.Sprintf("%sReq", upperName))
		reqTs, err := jsonToTypeScriptInterface(reqBodyStr, reqName)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, reqTs)
	}

	if resBodyStr != "" {
		resName = strcase.ToCamel(method + fmt.Sprintf("%sRes", upperName))
		resTs, err := jsonToTypeScriptInterface(resBodyStr, resName)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, resTs)
	}

	mtd := generateMethod(path, name, method, reqName, queryName, resName)
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
