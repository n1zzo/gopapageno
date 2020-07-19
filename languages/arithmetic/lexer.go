package arithmetic

import (
	"time"
	"unsafe"
    "fmt"
)

const (
	_ERROR       = -1
	_END_OF_FILE = 0
	_LEX_CORRECT = 1
	_SKIP        = 2
)

/*
lexer contains the file data and the current position.
*/
type lexer struct {
	data []byte
	pos  int
}

/*
yyLex reads a token from the lexer and creates a new symbol, saving it in genSym.
It returns one of the following codes:
ERROR       if a token couldn't be read
END_OF_FILE if there's no more data to lex
LEX_CORRECT if a token was successfully read
*/
func (l *lexer) yyLex(thread int, genSym *symbol) int {
	result := _SKIP
	for result == _SKIP {
		var lastFinalStateReached *lexerDfaState = nil
		var lastFinalStatePos int
		startPos := l.pos
		curState := &lexerAutomaton[0]
		for true {
			if l.pos == len(l.data) {
				return _END_OF_FILE
			}

			curStateIndex := curState.Transitions[l.data[l.pos]]

			//Cannot read chars anymore
			if curStateIndex == -1 {
				if lastFinalStateReached == nil {
					return _ERROR
				} else {
					l.pos = lastFinalStatePos + 1
					ruleNum := lastFinalStateReached.AssociatedRules[0]
					textBytes := l.data[startPos:l.pos]
					//TODO should be changed to safe code when Go supports no-op []byte to string conversion
					text := *(*string)(unsafe.Pointer(&textBytes))
					//fmt.Printf("%s: %d\n", text, ruleNum)
					result = lexerFunction(thread, ruleNum, text, genSym)
					break
				}
			} else {
				curState = &lexerAutomaton[curStateIndex]
				if curState.IsFinal {
					lastFinalStateReached = curState
					lastFinalStatePos = l.pos
					if l.pos == len(l.data)-1 {
						l.pos = lastFinalStatePos + 1
						ruleNum := lastFinalStateReached.AssociatedRules[0]
						textBytes := l.data[startPos:l.pos]
						//TODO should be changed to safe code when Go supports no-op []byte to string conversion
						text := *(*string)(unsafe.Pointer(&textBytes))
						//fmt.Printf("%s: %d\n", text, ruleNum)
						result = lexerFunction(thread, ruleNum, text, genSym)
						break
					}
				}
			}
			l.pos++
		}
	}
	return result
}

/*
Returns the index of the first cut point, according to the regular expression
specified in the lexer file.
*/
func findCutPoint(data []byte) int {
		startPos, curPos := 0, 0
        curState := &cutPointsAutomaton[0]
		for !curState.IsFinal {
			if curPos >= len(data) {
                startPos = 0
                break
			}
			curStateIndex := curState.Transitions[data[curPos]]
			if curStateIndex == -1 {
				startPos = curPos + 1
				curState = &cutPointsAutomaton[0]
			} else {
				curState = &lexerAutomaton[curStateIndex]
			}
			curPos++
        }
        return startPos
}

/*
lex is the lexing function executed in parallel by each thread.
It takes as input a lexThreadContext and a channel where it eventually sends
the result in form of a listOfStacks containing the lexed symbols.
*/
func lex(threadNum, numThreads int, data []byte, pool *stackPool, c chan lexResult) {
    //fmt.Printf("Threads #%d: before boundary check:", threadNum)
    //fmt.Printf("%s|\n", data)

    /* To preserve finite tokens boundaries, if we are not the first chunk,
       we look for delimiters, to skip any fragment of finite token */
    if threadNum != 0 {
        boundary := findCutPoint(data[:lexerLookahead+1])
        data = data[boundary:]
    }
    /* Conversely if we are not the last chunk, we search for the last
       delimiter in the lookahead section */
    if threadNum != numThreads-1 {
        boundary := findCutPoint(data[len(data) - lexerLookahead:]) + len(data) - lexerLookahead
        data = data[:boundary]
    }
    //fmt.Printf("Thread #%d: after boundary check:", threadNum)
    //fmt.Printf("%s|\n", data)

	start := time.Now()

	los := newLos(pool)

	sym := symbol{}

	lexer := lexer{data, 0}

	//Lex the first symbol
	res := lexer.yyLex(threadNum, &sym)

	//Keep lexing until the end of the file is reached or an error occurs
	for res != _END_OF_FILE {
		if res == _ERROR {
			c <- lexResult{threadNum, &los, false}
			return
		}
		los.Push(&sym)

		res = lexer.yyLex(threadNum, &sym)
	}

	c <- lexResult{threadNum, &los, true}

	Stats.LexTimes[threadNum] = time.Since(start)
}
