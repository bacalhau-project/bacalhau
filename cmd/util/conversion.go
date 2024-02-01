package util

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func YamlToJSON(data []byte) (string, error) {
	var body interface{}
	var res string

	if err := yaml.Unmarshal(data, &body); err != nil {
		return "", err
	}

	body = convert(body)

	if b, err := json.Marshal(body); err != nil {
		return "", err
	} else {
		res = string(b)
	}

	return res, nil
}

func JSONToYaml(data []byte) (string, error) {
	var body interface{}
	var res string

	if err := json.Unmarshal(data, &body); err != nil {
		return "", err
	}

	body = convert(body)

	if b, err := yaml.Marshal(body); err != nil {
		return "", err
	} else {
		res = string(b)
	}

	return res, nil
}

// find any `map[interface{}]interface{}` and convert them to `map[string]interface{}`
func convert(obj interface{}) interface{} {
	switch newObj := obj.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range newObj {
			m[k.(string)] = convert(v)
		}
		return m
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, v := range newObj {
			m[k] = convert(v)
		}
		return m
	case []interface{}:
		for i, v := range newObj {
			newObj[i] = convert(v)
		}
	}
	return obj
}
