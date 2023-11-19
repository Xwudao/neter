package typex

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func jsonToTypeScriptInterface(jsonStr string, interfaceName string) (string, error) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		return "", err
	}

	var typeScriptCode strings.Builder
	typeScriptCode.WriteString(fmt.Sprintf("export interface %s {\n", interfaceName))
	generateTypeScriptCode(&typeScriptCode, jsonData)
	typeScriptCode.WriteString("}\n")

	return typeScriptCode.String(), nil
}

func generateTypeScriptCode(code *strings.Builder, data map[string]interface{}) {
	for key, value := range data {
		code.WriteString(fmt.Sprintf("  %s: ", key))

		switch value.(type) {
		case string:
			code.WriteString("string;\n")
		case float64:
			code.WriteString("number;\n")
		case bool:
			code.WriteString("boolean;\n")
		case map[string]interface{}:
			code.WriteString("{\n")
			generateTypeScriptCode(code, value.(map[string]interface{}))
			code.WriteString("  };\n")
		case []interface{}:
			code.WriteString("Array<")
			if len(value.([]interface{})) > 0 {
				switch value.([]interface{})[0].(type) {
				case string:
					code.WriteString("string>;\n")
				case float64:
					code.WriteString("number>;\n")
				case bool:
					code.WriteString("boolean>;\n")
				case map[string]interface{}:
					code.WriteString("{\n")
					generateTypeScriptCode(code, value.([]interface{})[0].(map[string]interface{}))
					code.WriteString("  }>;\n")
				}
			} else {
				code.WriteString("any>;\n")
			}
		case nil:
			code.WriteString("null;\n")
		}
	}
}

func query2Map(queryString string) (map[string]interface{}, error) {
	// 创建一个 map 来存储查询参数
	queryParams := make(map[string]interface{})

	queryValues, err := url.ParseQuery(queryString)
	if err != nil {
		return queryParams, err
	}

	// 遍历 URL.Values 并处理多个值的情况
	for key, values := range queryValues {
		if len(values) == 1 {
			// 尝试将值转换为数字
			if num, err := strconv.ParseFloat(values[0], 64); err == nil {
				// 如果可以成功转换为数字，则存储数字
				queryParams[key] = num
			} else if boolValue, err := strconv.ParseBool(values[0]); err == nil {
				// 如果可以成功转换为布尔值，则存储布尔值
				queryParams[key] = boolValue
			} else {
				// 否则，存储字符串值
				queryParams[key] = values[0]
			}
		} else {
			// 如果有多个值，将它们保存为切片
			queryParams[key] = values
		}
	}

	return queryParams, nil
}

func query2JsonStr(queryString string) (string, error) {
	queryParams, err := query2Map(queryString)
	if err != nil {
		return "", err
	}
	// 将map转换为JSON字符串
	jsonData, err := json.Marshal(queryParams)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}
