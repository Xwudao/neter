package tsx

import (
	"encoding/json"
	"fmt"
	"net/url"
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
		}
	}
}
func query2JsonStr(queryString string) (string, error) {
	// 将查询字符串解析为URL.Values
	queryValues, err := url.ParseQuery(queryString)
	if err != nil {
		return "", err
	}

	// 创建一个map来存储查询参数
	queryParams := make(map[string]interface{})

	// 遍历URL.Values并处理多个值的情况
	for key, values := range queryValues {
		if len(values) == 1 {
			queryParams[key] = values[0]
		} else {
			// 如果有多个值，将它们保存为切片
			queryParams[key] = values
		}
	}

	// 将map转换为JSON字符串
	jsonData, err := json.Marshal(queryParams)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

//func main() {
//	jsonStr := `{
//		"name": "John",
//		"age": 30,
//		"active": true,
//		"address": {
//			"street": "123 Main St",
//			"city": "Anytown"
//		},
//		"arr": [
//			{
//				"age": 12,
//				"name": "12"
//			},
//			{
//				"age": 13,
//				"name": "13"
//			}
//		]
//	}`
//	interfaceName := "Person"
//
//	tsInterface := jsonToTypeScriptInterface(jsonStr, interfaceName)
//	fmt.Println(tsInterface)
//}
