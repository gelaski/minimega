package minicli

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type inputLexer struct {
	s        *bufio.Scanner
	items    []inputItem
	terminal string
	content  string
}

type inputItem struct {
	Value string
}

func printInput(items []inputItem) string {
	parts := make([]string, len(items))
	for i, v := range items {
		if strings.Contains(v.Value, ` `) {
			parts[i] = strconv.Quote(v.Value)
		} else {
			parts[i] = v.Value
		}
	}

	return strings.Join(parts, " ")
}

func lexInput(input string) ([]inputItem, error) {
	s := bufio.NewScanner(strings.NewReader(input))
	s.Split(bufio.ScanRunes)
	l := inputLexer{s: s, items: make([]inputItem, 0)}

	if err := l.Run(); err != nil {
		return nil, err
	}

	return l.items, nil
}

func (l *inputLexer) Run() (err error) {
	for state := l.lexOutside; state != nil && err == nil; {
		state, err = state()
	}

	return err
}

// lexOutside is our starting state. When we're in this state, we look for the
// start of a quote string or regular strings.
func (l *inputLexer) lexOutside() (stateFn, error) {
	emitContent := func() {
		// Emit item from processed content, if non-empty
		if len(l.content) > 0 {
			l.items = append(l.items, inputItem{Value: l.content})
			l.content = ""
		}
	}

	for l.s.Scan() {
		token := l.s.Text()
		switch token {
		case `"`, `'`:
			l.terminal = token
			return l.lexQuote, nil
		default:
			// Found the end of a string literal
			r, _ := utf8.DecodeRuneInString(token)
			if unicode.IsSpace(r) {
				emitContent()
				return l.lexOutside, nil
			}

			l.content += token
		}
	}

	emitContent()

	// Finished parsing pattern with no errors... Yippie kay yay
	return nil, nil
}

// lexQuote is the state where we've encountered a " or ' and we are scanning
// for the terminating " or '.
func (l *inputLexer) lexQuote() (stateFn, error) {
	// Scan until EOF, checking each token
	for l.s.Scan() {
		token := l.s.Text()
		switch token {
		case l.terminal:
			return l.lexOutside, nil
		default:
			l.content += token
		}
	}

	// We must have hit EOF before changing state
	return nil, fmt.Errorf("missing terminal %s", l.terminal)
}
