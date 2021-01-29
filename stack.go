package main

type Stack []interface{}

// IsEmpty: check if stack is empty
func (s *Stack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *Stack) Push(v interface{}) {
	*s = append(*s, v)
}

func (s *Stack) Pop() (interface{}, bool) {
	if s.IsEmpty() {
		return "", false
	}

	index := len(*s) - 1
	element := (*s)[index]
	*s = (*s)[:index]
	return element, true
}

func (s *Stack) Top() interface{} {
	if s.IsEmpty() {
		return ""
	}
	return (*s)[len(*s)-1]
}
