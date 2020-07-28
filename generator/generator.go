package generator

import (
	"fmt"

	"github.com/simoneguidi94/gopapageno/generator/regex"
)

func Generate(lexerFilename string, parserFilename string, outdir string) {
	lexRules, cutPoints, lexCode := parseLexer(lexerFilename)

	fmt.Printf("Lex rules (%d):\n", len(lexRules))
	for _, r := range lexRules {
		fmt.Println(r)
	}

	fmt.Printf("Cut points regex: %s\n", cutPoints)

	fmt.Println("Lex code:")
	fmt.Println(lexCode)

	var finiteDfa regex.Dfa
	var infiniteDfa []regex.Dfa

    /* Split between finite and infinite rules, to generate separate
       automata, for the infinite rules automata generate prefix and suffix
       accepting automata */
    var finiteRules, infiniteRules []lexRule
    for _, rule := range lexRules  {
        if isFinite(rule.Regex) {
            finiteRules = append(finiteRules, rule)
        } else {
            infiniteRules = append(infiniteRules, rule)
        }
    }

	if len(finiteRules) > 0 {
		var nfa *regex.Nfa
		for i := 0; i < len(finiteRules); i++ {
			var curNfa *regex.Nfa
            success, result := regex.ParseString([]byte(finiteRules[i].Regex), 1)
			if success {
                if (i == 0) {
                    nfa = result.Value.(*regex.Nfa)
				    nfa.AddAssociatedRule(i)
                } else {
				    curNfa = result.Value.(*regex.Nfa)
				    curNfa.AddAssociatedRule(i)
				    nfa.Unite(*curNfa)
                }
			} else {
				fmt.Println("Error: could not parse the following regular expression:", finiteRules[i].Regex)
				return
			}
		}

		finiteDfa = nfa.ToDfa()

		/*ok, hasRuleNum, ruleNum := dfa.Check([]byte(" "))
		if ok {
			fmt.Println("Ok")
			if hasRuleNum {
				fmt.Println("RuleNum:", ruleNum)
			} else {
				fmt.Println("No rule")
			}
		} else {
			fmt.Println("Not ok")
		}*/
	} else {
		fmt.Println("Error: the lexer does not contain any rule")
		return
	}

    // TODO: add support for prefix and suffix matching
    if len(infiniteRules) > 0 {
        for _, rule := range infiniteRules {
		    var nfa *regex.Nfa
            success, result := regex.ParseString([]byte(rule.Regex), 1)
		    if success {
                nfa = result.Value.(*regex.Nfa)
			    nfa.AddAssociatedRule(0)
			} else {
			    fmt.Println("Error: could not parse the following regular expression:", rule.Regex)
			    return
			}

		    infiniteDfa = append(infiniteDfa, nfa.ToDfa())
        }
    }

	var cutPointsDfa regex.Dfa
	if cutPoints == "" {
		cutPointsNfa := regex.NewEmptyStringNfa()
		cutPointsDfa = cutPointsNfa.ToDfa()
	} else {
		var cutPointsNfa *regex.Nfa
		success, result := regex.ParseString([]byte(cutPoints), 1)
		if success {
			cutPointsNfa = result.Value.(*regex.Nfa)
		} else {
			fmt.Println("Error: could not parse the following regular expression:", cutPoints)
			return
		}
		cutPointsDfa = cutPointsNfa.ToDfa()
	}

	parserPreamble, axiom, rules := parseGrammar(parserFilename)

	fmt.Println("Go preamble:")
	fmt.Println(parserPreamble)

	if axiom == "" {
		fmt.Println("Error: the axiom is not defined")
		return
	} else {
		fmt.Println("Axiom:", axiom)
	}

	fmt.Printf("Rules (%d):\n", len(rules))
	for _, r := range rules {
		fmt.Println(r)
	}

	nonterminals, terminals := inferTokens(rules)

	fmt.Printf("Nonterminals (%d): %s\n", len(nonterminals), nonterminals)
	fmt.Printf("Terminals (%d): %s\n", len(terminals), terminals)

	if !checkAxiomUsage(rules, axiom) {
		fmt.Println("Error: the axiom isn't used in any rule")
		return
	}

	newRules, newNonterminals := deleteRepeatedRHS(nonterminals, terminals, axiom, rules)

	fmt.Printf("New rules after elimination of repeated rhs (%d):\n", len(newRules))
	for _, r := range newRules {
		fmt.Println(r)
	}

	fmt.Printf("New nonterminals (%d): %s\n", len(newNonterminals), newNonterminals)

	precMatrix, err := createPrecMatrix(newRules, newNonterminals, terminals)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Precedence matrix:")
	fmt.Println(precMatrix)

	sortedRules := sortRulesByRHS(newRules, newNonterminals, terminals)
	fmt.Printf("Sorted rules (%d):\n", len(sortedRules))
	for _, r := range sortedRules {
		fmt.Println(r)
	}

	err = emitOutputFolder(outdir)
	handleEmissionError(err)
	err = emitLexerFunction(outdir, lexCode, lexRules)
	handleEmissionError(err)
	err = emitLexerAutomata(outdir, finiteDfa, infiniteDfa, cutPointsDfa)
	handleEmissionError(err)
	err = emitTokens(outdir, newNonterminals, terminals)
	handleEmissionError(err)
	err = emitRules(outdir, sortedRules, newNonterminals, terminals)
	handleEmissionError(err)
	err = emitFunction(outdir, parserPreamble, sortedRules)
	handleEmissionError(err)
	err = emitPrecMatrix(outdir, terminals, precMatrix)
	handleEmissionError(err)
	err = emitCommonFiles(outdir)
	handleEmissionError(err)
}

func handleEmissionError(e error) {
	if e != nil {
		fmt.Println(e.Error())
	}
}
