package errors

import "strings"

type ErrorCollector []error

func (c *ErrorCollector) Collect(e error) {
	if e != nil {
		*c = append(*c, e)
	}
}

func (c *ErrorCollector) Error() (errorString string) {
	messages := []string{}
	for i := range *c {
		messages = append(messages, (*c)[i].Error())
	}
	return strings.Join(messages, "\n")
}
