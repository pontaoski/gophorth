package main

type AnonStack []Stack

// IsEmpty: check if stack is empty
func (s *AnonStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *AnonStack) Push(v Stack) {
	*s = append(*s, v)
}

func (s *AnonStack) Pop() (Stack, bool) {
	if s.IsEmpty() {
		return Stack{}, false
	}

	index := len(*s) - 1
	element := (*s)[index]
	*s = (*s)[:index]
	return element, true
}

func (s *AnonStack) Top() Stack {
	if s.IsEmpty() {
		return nil
	}
	return (*s)[len(*s)-1]
}

func (s *AnonStack) PushOnTop(v interface{}) {
	(*s)[len(*s)-1].Push(v)
}
