package main

import "encoding/json"

func Stringify(list []Release) string {
	var context []Context
	for _, r := range list {

		context = append(context, Context{
			Body:        r.Body,
			CreatedDate: r.CreatedAt.String(),
			HtmlURL:     r.HTMLURL,
		})
	}
	return ToString(PromptContext{Context: context})
}

func ToString(context PromptContext) string {
	jsonData, err := json.Marshal(context)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
