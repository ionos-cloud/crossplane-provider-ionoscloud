package substitution

// GlobalState defines the global state of the substitutions
// it's a map of index to a list of substitutions
type GlobalState map[Identifier][]State

func NewGlobalState() *GlobalState {
	return &GlobalState{}
}

// Each iterates over the global state and calls the provided function
func (gs GlobalState) Each(f func(Identifier, []State)) {
	for k, v := range gs {
		f(k, v)
	}
}

// Set adds a new state to the global state
func (gs GlobalState) Set(identifier Identifier, key string, value string) {
	if gs[identifier] == nil {
		gs[identifier] = []State{}
	}

	gs[identifier] = append(gs[identifier], State{
		Key:   key,
		Value: value,
	})
}

// GetByIdentifier returns the state by the provided identifier
func (gs GlobalState) GetByIdentifier(identifier Identifier) []State {
	state, ok := gs[identifier]
	if !ok {
		gs[identifier] = []State{}
		return gs[identifier]
	}

	return state
}

// Len returns the length of the global state
func (gs GlobalState) Len() int {
	return len(gs)
}

// Exists checks if a key exists in the global state
func (gs GlobalState) Exists(identifier Identifier, key string) bool {
	state := gs.GetByIdentifier(identifier)
	for _, s := range state {
		if s.Key == key {
			return true
		}
	}

	return false
}

// State keeps track of "Key" to generated "Value".
type State struct {
	Key   string
	Value string
}
