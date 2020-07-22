package regex

import "fmt"

const _EPSILON = 0

type NfaState struct {
	Transitions     [256][]*NfaState
	AssociatedRules []int
}

func (state *NfaState) EpsilonClosure() []*NfaState {
	closure := make([]*NfaState, 0)

	closure = append(closure, state)

	nextStateToCheckPos := 0

	for nextStateToCheckPos < len(closure) {
		curState := closure[nextStateToCheckPos]
		closure = append(closure, curState.Transitions[_EPSILON]...)
		nextStateToCheckPos++
	}

	return closure
}

func stateSetContains(stateSet []*NfaState, state *NfaState) bool {
	for _, curState := range stateSet {
		if curState == state {
			return true
		}
	}
	return false
}

func stateSetEpsilonClosure(stateSet []*NfaState) []*NfaState {
	closure := make([]*NfaState, 0, len(stateSet))
	for _, curState := range stateSet {
		curEpsilonClosure := curState.EpsilonClosure()

		for _, curClosureState := range curEpsilonClosure {
			if !stateSetContains(closure, curClosureState) {
				closure = append(closure, curClosureState)
			}
		}
	}

	return closure
}

func stateSetMove(stateSet []*NfaState, char int) []*NfaState {
	states := make([]*NfaState, 0)

	for _, curState := range stateSet {
		reachedStates := curState.Transitions[char]

		for _, curReachedState := range reachedStates {
			if !stateSetContains(states, curReachedState) {
				states = append(states, curReachedState)
			}
		}
	}

	return states
}

func (state *NfaState) AddTransition(char byte, ptr *NfaState) {
	if state.Transitions[char] == nil {
		state.Transitions[char] = []*NfaState{ptr}
	} else {
		state.Transitions[char] = append(state.Transitions[char], ptr)
	}
}

type Nfa struct {
	Initial   *NfaState
	Final     *NfaState
	NumStates int
}

func NewEmptyStringNfa() Nfa {
	nfa := Nfa{}
	nfaInitialFinal := NfaState{}

	nfa.Initial = &nfaInitialFinal
	nfa.Final = &nfaInitialFinal

	nfa.NumStates = 1

	return nfa
}

func newNfaFromChar(char byte) Nfa {
	nfa := Nfa{}
	nfaInitial := NfaState{}
	nfaFinal := NfaState{}

	nfaInitial.AddTransition(char, &nfaFinal)
	nfa.Initial = &nfaInitial
	nfa.Final = &nfaFinal

	nfa.NumStates = 2

	return nfa
}

func newNfaFromCharClass(chars [256]bool) Nfa {
	nfa := Nfa{}
	nfaInitial := NfaState{}
	nfaFinal := NfaState{}

	for i, thereIs := range chars {
		if thereIs {
			nfaInitial.AddTransition(byte(i), &nfaFinal)
		}
	}

	nfa.Initial = &nfaInitial
	nfa.Final = &nfaFinal

	nfa.NumStates = 2

	return nfa
}

func newNfaFromString(str []byte) Nfa {
	nfa := Nfa{}
	firstState := NfaState{}

	curState := &firstState

	nfa.Initial = curState

	for _, curChar := range str {
		newState := NfaState{}
		curState.AddTransition(curChar, &newState)
		curState = &newState
	}

	nfa.Final = curState

	nfa.NumStates = len(str) + 1

	return nfa
}

func (nfa1 *Nfa) Concatenate(nfa2 Nfa) {
	*nfa1.Final = *nfa2.Initial
	nfa1.Final = nfa2.Final

	nfa1.NumStates = nfa1.NumStates + nfa2.NumStates - 1
}

//Operator |
func (nfa1 *Nfa) Unite(nfa2 Nfa) {
	newInitial := NfaState{}
	newFinal := NfaState{}

	newInitial.AddTransition(_EPSILON, nfa1.Initial)
	newInitial.AddTransition(_EPSILON, nfa2.Initial)

	nfa1.Final.AddTransition(_EPSILON, &newFinal)
	nfa2.Final.AddTransition(_EPSILON, &newFinal)

	nfa1.Initial = &newInitial
	nfa1.Final = &newFinal

	nfa1.NumStates += nfa2.NumStates + 2
}

//Operator *
func (nfa *Nfa) KleeneStar() {
	newInitial := NfaState{}
	newFinal := NfaState{}

	newInitial.AddTransition(_EPSILON, nfa.Initial)
	newInitial.AddTransition(_EPSILON, &newFinal)

	nfa.Final.AddTransition(_EPSILON, nfa.Initial)
	nfa.Final.AddTransition(_EPSILON, &newFinal)

	nfa.Initial = &newInitial
	nfa.Final = &newFinal

	nfa.NumStates += 2
}

//Operator +
func (nfa *Nfa) KleenePlus() {
	newInitial := NfaState{}
	newFinal := NfaState{}

	newInitial.AddTransition(_EPSILON, nfa.Initial)

	nfa.Final.AddTransition(_EPSILON, nfa.Initial)
	nfa.Final.AddTransition(_EPSILON, &newFinal)

	nfa.Initial = &newInitial
	nfa.Final = &newFinal

	nfa.NumStates += 2
}

//Operator ?
func (nfa *Nfa) ZeroOrOne() {
	newInitial := NfaState{}
	newFinal := NfaState{}

	newInitial.AddTransition(_EPSILON, nfa.Initial)
	newInitial.AddTransition(_EPSILON, &newFinal)

	nfa.Final.AddTransition(_EPSILON, &newFinal)

	nfa.Initial = &newInitial
	nfa.Final = &newFinal

	nfa.NumStates += 2
}

func (nfa *Nfa) AddAssociatedRule(ruleNum int) {
	finalState := nfa.Final

	if finalState.AssociatedRules == nil {
		finalState.AssociatedRules = []int{ruleNum}
	} else {
		finalState.AssociatedRules = append(finalState.AssociatedRules, ruleNum)
	}
}

func Visit(nfa *NfaState, visited map[*NfaState]bool) {
	visited[nfa] = true
	for _, t := range nfa.Transitions {
		for _, n := range t {
			if n != nil && !visited[n] {
				Visit(n, visited)
			}
		}
	}
}

func SelectFinals(final *NfaState, visited map[*NfaState]bool, finals map[*NfaState]bool) {
	/* To compute the set of final states, iteratively collect all the states
	   which have epsilon transitions to the final states or to states
	   belonging to the aforementioned set. */
	final_count := -1
	for final_count != len(finals) {
		final_count = len(finals)
		for n := range visited {
			if n == final {
				finals[n] = true
				continue
			}
			for _, t := range n.Transitions[_EPSILON] {
				if t == final || finals[t] {
					finals[n] = true
					break
				}
			}
		}
	}
}

func Subtract(x map[*NfaState]bool, y map[*NfaState]bool) {
	for n := range x {
		if y[n] {
			delete(x, n)
		}
	}
}

func (nfa *Nfa) ToPrefix() {
	/* To accept prefixes, create a new final state with epsilon-transitions
	   to every other state except the initial state */
	oldInitial := nfa.Initial
	visited_nfa := make(map[*NfaState]bool)
	Visit(oldInitial, visited_nfa)
	/* Skip the first state, or states reachables only through epsilon
	   transitions, to avoid accepting empty strings */
	_, ok := visited_nfa[oldInitial];
	if ok {
		delete(visited_nfa, oldInitial);
	}

	/* Check that the final state have at least one associated rule */
	if len(nfa.Final.AssociatedRules) == 0 {
		fmt.Println("Error, making final a state without associated rules!")
	}

	newNfaState := &NfaState{}
	for n := range visited_nfa {
		n.Transitions[_EPSILON] = append(n.Transitions[_EPSILON], newNfaState)
	}
	newNfaState.AssociatedRules = nfa.Final.AssociatedRules
	nfa.Final = newNfaState
	nfa.NumStates += 1
}

func (nfa *Nfa) ToSuffix() {
	/* To accept suffixes, create a new initial state with epsilon-transitions
	 * to every other state, except final states */
	visited := make(map[*NfaState]bool)
	Visit(nfa.Initial, visited)
	/* Exclude final state when building epsilon-transitions, which are
	   final states or states with epsilon transitions to a final state */
	finals := make(map[*NfaState]bool)
	SelectFinals(nfa.Final, visited, finals)
	Subtract(visited, finals)
	visited_slice := make([]*NfaState, 0, len(visited))
	for s := range visited {
		visited_slice = append(visited_slice, s)
	}
	/* If the resulting set is empty, use the initial state */
	if len(visited_slice) == 0 {
		visited_slice = append(visited_slice, nfa.Initial)
	}
	newNfaState := &NfaState{}
	newNfaState.AssociatedRules = nfa.Initial.AssociatedRules
	newNfaState.Transitions[_EPSILON] = visited_slice
	nfa.Initial = newNfaState
	nfa.NumStates += 1
}

func (nfa *Nfa) ToPrefixSuffix() {
	nfa.ToPrefix()
	nfa.ToSuffix()
}

func (nfa *Nfa) ToDfa() Dfa {
	genStates := make([]nfaStateSetPtr, 0)

	curDfaStateNum := 0

	initialDfaState := DfaState{}
	initialDfaState.Num = curDfaStateNum

	dfa := Dfa{&initialDfaState, make([]*DfaState, 0), 1}

	genStates = append(genStates, nfaStateSetPtr{nfa.Initial.EpsilonClosure(), &initialDfaState})

	search := func(gStates []nfaStateSetPtr, stateSet []*NfaState) *nfaStateSetPtr {
		for _, curGState := range gStates {
			if len(curGState.StateSet) != len(stateSet) {
				continue
			}
			equal := true
			for i, _ := range curGState.StateSet {
				if curGState.StateSet[i] != stateSet[i] {
					equal = false
					break
				}
			}
			if equal {
				return &curGState
			}
		}
		return nil
	}

	nextStateToCheckPos := 0

	for nextStateToCheckPos < len(genStates) {
		curStateSet := genStates[nextStateToCheckPos].StateSet
		curDfaState := genStates[nextStateToCheckPos].Ptr

		//For each character
		for i := 1; i < 256; i++ {
			charStateSet := stateSetMove(curStateSet, i)
			epsilonClosure := stateSetEpsilonClosure(charStateSet)

			if len(epsilonClosure) == 0 {
				continue
			}

			foundStateSetPtr := search(genStates, epsilonClosure)

			if foundStateSetPtr != nil {
				curDfaState.Transitions[i] = foundStateSetPtr.Ptr
			} else {
				curDfaStateNum++
				newDfaState := DfaState{}
				newDfaState.Num = curDfaStateNum
				newDfaState.AssociatedRules = make([]int, 0)
				for _, curNfaState := range epsilonClosure {
					newDfaState.AssociatedRules = append(newDfaState.AssociatedRules, curNfaState.AssociatedRules...)
				}
				curDfaState.Transitions[i] = &newDfaState
				newStateSetPtr := nfaStateSetPtr{epsilonClosure, &newDfaState}

				genStates = append(genStates, newStateSetPtr)

				// If the state set contains a final state, make it final
				if stateSetContains(newStateSetPtr.StateSet, nfa.Final) {
					newStateSetPtr.Ptr.IsFinal = true
					dfa.Final = append(dfa.Final, newStateSetPtr.Ptr)
				}
			}
		}
		nextStateToCheckPos++
	}

	dfa.NumStates = len(genStates)

	return dfa
}

type DfaState struct {
	Num             int
	Transitions     [256]*DfaState
	IsFinal         bool
	AssociatedRules []int
}

type Dfa struct {
	Initial   *DfaState
	Final     []*DfaState
	NumStates int
}

func (dfaState *DfaState) getStatesR(addedStates *[]*DfaState) {
	//The state was already added, return
	if (*addedStates)[dfaState.Num] != nil {
		return
	}
	(*addedStates)[dfaState.Num] = dfaState

	for _, nextState := range dfaState.Transitions {
		if nextState != nil {
			nextState.getStatesR(addedStates)
		}
	}
}

/*
GetState returns a slice containing all the states of the dfa.
The states are sorted by their state number.
*/
func (dfa *Dfa) GetStates() []*DfaState {
	states := make([]*DfaState, dfa.NumStates)

	dfa.Initial.getStatesR(&states)

	return states
}

/*func (dfa *Dfa) Check(str []byte) (bool, bool, uint16) {
	curState := dfa.Initial

	//fmt.Println(curState)

	for _, curChar := range str {
		curState = curState.Transitions[curChar]

		//fmt.Println(curState)

		if curState == nil {
			return false, false, 0
		}
	}

	if len(curState.AssociatedTokens) == 0 {
		return curState.IsFinal, false, 0
	}

	index := 0
	minRule := curState.AssociatedTokens[0].RuleNum

	for i := 1; i < len(curState.AssociatedTokens); i++ {
		if curState.AssociatedTokens[i].RuleNum < minRule {
			minRule = curState.AssociatedTokens[i].RuleNum
			index = i
		}
	}

	return curState.IsFinal, true, curState.AssociatedTokens[index].Token
}*/

type nfaStateSetPtr struct {
	StateSet []*NfaState
	Ptr      *DfaState
}

func (stateSet1 *nfaStateSetPtr) Equals(stateSet2 *nfaStateSetPtr) bool {
	if len(stateSet1.StateSet) != len(stateSet2.StateSet) {
		return false
	}
	for i, _ := range stateSet1.StateSet {
		if stateSet1.StateSet[i] != stateSet2.StateSet[i] {
			return false
		}
	}
	return true
}

/* Post-order DFS tree exploration */
func (dfaState *DfaState) ToNfaState(count *int,
									 final *NfaState,
									 nodes []*NfaState) *NfaState {
	node := nodes[dfaState.Num]
	if node != nil {
		return node
	}
	*count += 1
	nfaState := NfaState{}
	nfaState.AssociatedRules = dfaState.AssociatedRules
	nodes[dfaState.Num] = &nfaState
	if dfaState.IsFinal {
		nfaState.Transitions[_EPSILON] = []*NfaState{final}
	}
	for i := 0; i < 256; i++ {
		next := dfaState.Transitions[i]
		if next != nil {
			nfaState.Transitions[i] = []*NfaState{next.ToNfaState(count,
																  final,
																  nodes)}
		}
	}
	return &nfaState
}

/* Convert deterministic automata into non-deterministic one, to be able
   to simulate many NFA with a lower memory footprint */
func (dfa *Dfa) ToNfa() *Nfa {
	nfa := Nfa{}
	nodes := make([]*NfaState, dfa.NumStates)
	final := NfaState{}
	nfa.Final = &final
	nfa.Initial = dfa.Initial.ToNfaState(&nfa.NumStates, &final, nodes)
	return &nfa
}
