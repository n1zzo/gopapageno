package main

import (
    "fmt"

    "github.com/n1zzo/gopapageno/generator/regex"
)

const (
    _ERROR       = -1
    _END_OF_FILE = 0
    _LEX_CORRECT = 1
    _SKIP        = 2
)

type lexerDfaState struct {
    Transitions     [256]int
    IsFinal         bool
    AssociatedRules []int
}

type lexerDfa []lexerDfaState

func match(dfa regex.Dfa, s string) int {
    input := []byte(s)
    var lastFinalStateReached *regex.DfaState = nil
    var lastFinalStatePos int
    pos := 0
    curState := dfa.Initial
    for true {
        fmt.Println(pos)
        if pos == len(input) {
            return _END_OF_FILE
        }

        curState = curState.Transitions[input[pos]]

        // Cannot read chars anymore
        if curState == nil {
            if lastFinalStateReached == nil {
                return _ERROR
            } else {
                pos = lastFinalStatePos + 1
                fmt.Printf("Matched substring\n")
                return _ERROR
            }
        } else {
            if curState.IsFinal {
                lastFinalStateReached = curState
                lastFinalStatePos = pos
                if pos == len(s)-1 {
                    pos = lastFinalStatePos + 1
                    break
                }
            }
        }
        pos++
    }
    return _LEX_CORRECT
}

func main() {
    r := "abcdef"

    test := []string{"abcdef", "abcde", "bcdef", "bcd", "c", "ce"}
    result := []int {_LEX_CORRECT, _LEX_CORRECT, _LEX_CORRECT, _LEX_CORRECT,
                     _LEX_CORRECT, _ERROR}

    var nfa *regex.Nfa
    success, res := regex.ParseString([]byte(r), 1)
    if success {
        nfa = res.Value.(*regex.Nfa)
        nfa.AddAssociatedRule(0)
    } else {
        fmt.Println("Error: could not parse the following regular expression:", r)
        return
    }

    nfa.ToPrefixSuffix()
    dfa := nfa.ToDfa()

    pass := true
    for i, s := range test {
        fmt.Println("Parsing: ", s)
        res := match(dfa, s)
        if res != _LEX_CORRECT {
            fmt.Println("Error")
        } else {
            fmt.Println("Accepted")
        }
        if res != result[i] {
            pass = false
        }
    }
    if pass {
        fmt.Println("PASS")
    } else {
        fmt.Println("FAIL")
    }
}
