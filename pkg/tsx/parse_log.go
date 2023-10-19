package tsx

import (
	"fmt"
	"io"
	"os"
	"strings"

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
		//path       string
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
			//path = strings.TrimSpace(cnt)
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

	strcase.ConfigureAcronym("POST", "post")
	strcase.ConfigureAcronym("GET", "get")
	strcase.ConfigureAcronym("PUT", "put")
	strcase.ConfigureAcronym("DELETE", "delete")
	strcase.ConfigureAcronym("OPTIONS", "options")

	if query != "" {
		qJ, err := query2JsonStr(query)
		if err != nil {
			return nil, err
		}
		qTs, err := jsonToTypeScriptInterface(qJ, strcase.ToLowerCamel(method+fmt.Sprintf("%sQuery", name)))
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, qTs)
	}

	if reqBodyStr != "" {
		reqTs, err := jsonToTypeScriptInterface(reqBodyStr, strcase.ToLowerCamel(method+fmt.Sprintf("%sReq", name)))
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, reqTs)
	}

	if resBodyStr != "" {
		resTs, err := jsonToTypeScriptInterface(resBodyStr, strcase.ToLowerCamel(method+fmt.Sprintf("%sRes", name)))
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, resTs)
	}

	return rtn, nil
}
