package typex

import (
	"io"
	"os"
	"strings"
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

	if rtn.Query != "" {
		rtn.QueryMap, _ = query2Map(rtn.Query)
	}

	return &rtn, nil
}
