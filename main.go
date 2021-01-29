package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type interpreter struct {
	reader *bufio.Scanner

	words                  map[string]word
	stack                  Stack
	compiling              bool
	commenting             bool
	currentlyCompilingWord string
	currentlyCompiling     Stack
	anonymousStack         AnonStack
}

type word struct {
	native    func(i *interpreter)
	nonNative Stack
	immediate bool
}

type wordReference struct {
	from map[string]word
	key  string
}

type wordLiteral struct {
	items Stack
}

func (w word) run(i *interpreter) {
	if w.native != nil {
		w.native(i)
	} else {
		for _, item := range w.nonNative {
			i.ExecuteWord(stringify(item))
		}
	}
}

func stringify(i interface{}) string {
	switch item := i.(type) {
	case string:
		return item
	case int64:
		return strconv.FormatInt(item, 6)
	case wordLiteral:
		var items []string
		for _, item := range item.items {
			items = append(items, stringify(item))
		}
		return "[ " + strings.Join(items, " ") + " ]"
	case wordReference:
		return "& " + item.key
	}
	panic("unhandled")
}

func newInterpreter(r io.Reader) *interpreter {
	it := &interpreter{
		words: map[string]word{
			"print": {
				native: func(i *interpreter) {
					v := i.stack.Top()
					println(stringify(v))
				},
			},
			"stack": {
				native: func(i *interpreter) {
					for idx, item := range i.stack {
						println(idx, "\t"+stringify(item))
					}
				},
			},
			":": {
				native: func(i *interpreter) {
					word, err := i.ReadWord()
					if err != nil {
						log.Fatal(err)
					}
					i.currentlyCompilingWord = word
					i.compiling = true
				},
			},
			";": {
				native: func(i *interpreter) {
					i.words[i.currentlyCompilingWord] = word{
						nonNative: i.currentlyCompiling,
						immediate: false,
					}
					i.currentlyCompiling = Stack{}
					i.currentlyCompilingWord = ""
					i.compiling = false
				},
				immediate: true,
			},
			"+": {
				native: func(i *interpreter) {
					v, ok := i.stack.Pop()
					if !ok {
						panic(ok)
					}
					v2, ok := i.stack.Pop()
					if !ok {
						panic(ok)
					}
					i.stack.Push(v.(int64) + v2.(int64))
				},
			},
			"go": {
				native: func(i *interpreter) {
					word, err := i.ReadWord()
					if err != nil {
						log.Fatal(err)
					}

					go func() {
						interpreter := &interpreter{}
						interpreter.words = i.words
						interpreter.words[word].run(i)
					}()
				},
			},
			"sleep": {
				native: func(i *interpreter) {
					nanpa, ok := i.stack.Pop()
					if !ok {
						log.Fatalf("not a number")
					}
					val := nanpa.(int64)
					time.Sleep(time.Duration(val) * time.Second)
				},
			},
			"&": {
				native: func(i *interpreter) {
					word, err := i.ReadWord()
					if err != nil {
						log.Fatal(err)
					}

					i.stack.Push(wordReference{
						from: i.words,
						key:  word,
					})
				},
			},
			"[": {
				native: func(i *interpreter) {
					i.anonymousStack.Push(Stack{})
				},
			},
			"]": {
				native: func(i *interpreter) {
					val, ok := i.anonymousStack.Pop()
					if !ok {
						log.Fatal("popped from empty anonymous stack")
					}
					i.stack.Push(wordLiteral{
						items: val,
					})
				},
				immediate: true,
			},
			"dup": {
				native: func(i *interpreter) {
					i.stack.Push(i.stack.Top())
				},
			},
			"run": {
				native: func(i *interpreter) {
					items, ok := i.stack.Pop()
					if !ok {
						panic("not ok")
					}
					if v, ok := items.(wordReference); ok {
						v.from[v.key].run(i)
						return
					}
					for _, item := range items.(wordLiteral).items {
						i.ExecuteWord(stringify(item))
					}
				},
			},
			"pop": {
				native: func(i *interpreter) {
					i.stack.Pop()
				},
			},
			"if": {
				native: func(i *interpreter) {
					cond, ok := i.stack.Pop()
					if !ok {
						panic("not ok")
					}
					if cond.(int64) == 0 {
						return
					}

					exec, ok := i.stack.Pop()
					if !ok {
						panic("not ok")
					}

					if v, ok := exec.(wordReference); ok {
						v.from[v.key].run(i)
						return
					}
					for _, item := range exec.(wordLiteral).items {
						i.ExecuteWord(stringify(item))
					}
				},
			},
			"=": {
				native: func(i *interpreter) {
					lhs, ok := i.stack.Pop()
					if !ok {
						panic("not ok")
					}
					rhs, ok := i.stack.Pop()
					if !ok {
						panic("not ok")
					}
					if reflect.DeepEqual(lhs, rhs) {
						i.stack.Push(int64(1))
					} else {
						i.stack.Push(int64(0))
					}
				},
			},
			"!": {
				native: func(i *interpreter) {
					val, ok := i.stack.Pop()
					if !ok {
						panic("not ok")
					}
					if val.(int64) == 0 {
						i.stack.Push(int64(1))
					} else {
						i.stack.Push(int64(0))
					}
				},
			},
		},
		stack: Stack{},
	}

	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanWords)
	it.reader = scanner

	return it
}

func (i *interpreter) ReadWord() (string, error) {
	if ok := i.reader.Scan(); ok {
		return i.reader.Text(), nil
	}
	return "", i.reader.Err()
}

func (i *interpreter) ExecuteWord(s string) {
	if s == "(" {
		i.commenting = true
		return
	}
	if i.commenting {
		if s == ")" {
			i.commenting = false
		}
		return
	}
	if !i.anonymousStack.IsEmpty() {
		if val, ok := i.words[s]; ok {
			if val.immediate && s != "]" {
				panic("cannot use immediate word in closure")
			} else if val.immediate && s == "]" {
				val.run(i)
				return
			}

			val, err := strconv.ParseInt(s, 6, 64)
			if err == nil {
				i.anonymousStack.PushOnTop(val)
			} else {
				i.anonymousStack.PushOnTop(s)
			}
		} else {
			val, err := strconv.ParseInt(s, 6, 64)
			if err == nil {
				i.anonymousStack.PushOnTop(val)
			} else {
				log.Fatalf("bad word: %s", s)
			}
		}

		return
	}

	if val, ok := i.words[s]; ok {
		if !i.compiling || val.immediate {
			val.run(i)
		} else {
			val, err := strconv.ParseInt(s, 6, 64)
			if err == nil {
				i.currentlyCompiling.Push(val)
			} else {
				i.currentlyCompiling.Push(s)
			}
		}
	} else {
		val, err := strconv.ParseInt(s, 6, 64)
		if err == nil {
			i.stack.Push(val)
		} else {
			log.Fatalf("bad word: %s", s)
		}
	}

}

func main() {
	it := newInterpreter(os.Stdin)

	for {
		word, err := it.ReadWord()
		if err != nil {
			log.Fatal(err)
		}
		it.ExecuteWord(word)
	}
}
