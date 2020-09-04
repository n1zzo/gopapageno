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

    /* Split between finite and infinite rules, to generate separate
       automata, for the infinite rules automata generate prefix and suffix
       accepting automata */
    var finiteRules, infiniteRules []int
    for i, rule := range lexRules  {
        if isFinite(rule.Regex) {
            finiteRules = append(finiteRules, i)
        } else {
            infiniteRules = append(infiniteRules, i)
        }
    }

    fmt.Println("Finite Rules")
    fmt.Println(finiteRules)
    fmt.Println("Infinite Rules")
    fmt.Println(infiniteRules)

    // Rule numbering is: all the finite rules + all the infinite rules
    if len(finiteRules) <= 0 && len(infiniteRules) <= 0 {
		fmt.Println("Error: the lexer does not contain any rule")
		return
	}
    fmt.Println("Processing Finite NFA")
	var finiteNfa *regex.Nfa
    for i, n := range finiteRules {
		var curNfa *regex.Nfa
        success, result := regex.ParseString([]byte(lexRules[n].Regex), 1)
		if success {
            if (i == 0) {
                finiteNfa = result.Value.(*regex.Nfa)
			    finiteNfa.AddAssociatedRule(n)
            } else {
			    curNfa = result.Value.(*regex.Nfa)
			    curNfa.AddAssociatedRule(n)
			    finiteNfa.Unite(*curNfa)
            }
		} else {
			fmt.Println("Error: could not parse the following regular expression:", lexRules[n].Regex)
			return
		}
	}
    // One DFA for the finite tokens
    finiteDfa := finiteNfa.ToDfa()

    // One DFA for each infinite token
    var infDfa, prefixInfDfa, suffixInfDfa, prefixSuffixInfDfa []regex.Dfa
    for _, n := range infiniteRules {
        var infiniteNfa *regex.Nfa
        success, result := regex.ParseString([]byte(lexRules[n].Regex), 1)
	    if success {
            fmt.Println("Processing Infinite DFA")
		    infiniteNfa = result.Value.(*regex.Nfa)
		    infiniteNfa.AddAssociatedRule(n)
            nfa := infiniteNfa
		    infDfa = append(infDfa, infiniteNfa.ToDfa())
            fmt.Println("Processing Prefix DFA")
            infiniteNfa.ToPrefix()
		    prefixInfDfa = append(prefixInfDfa, infiniteNfa.ToDfa())
            fmt.Println("Processing Suffix DFA")
            nfa.ToSuffix()
            suffixInfDfa = append(suffixInfDfa, nfa.ToDfa())
            fmt.Println("Processing Prefix Suffix DFA")
            infiniteNfa.ToSuffix()
            prefixSuffixInfDfa = append(prefixSuffixInfDfa, infiniteNfa.ToDfa())
		} else {
		    fmt.Println("Error: could not parse the following regular expression:", lexRules[n].Regex)
		    return
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
	err = emitLexerAutomata(outdir,
                            finiteDfa,
                            infDfa,
                            prefixInfDfa,
                            suffixInfDfa,
                            prefixSuffixInfDfa,
                            cutPointsDfa)
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
