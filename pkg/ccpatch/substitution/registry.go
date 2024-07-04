package substitution

// Identifier is used to identify the state of the current replica
type Identifier string

// Handler defines the interface for a substitution handler
type Handler interface {
	Type() string
	WriteGlobalState(identifier Identifier, state *GlobalState, sub Substitution) error
}

var registeredSubstitutions = make(map[string]Handler)

// RegisterSubstitution registers a new substitution
func RegisterSubstitution(sub Handler) {
	if sub == nil {
		panic("cannot register a nil substitution")
	}

	if sub.Type() == "" {
		panic("cannot register a substitution with an empty type")
	}

	registeredSubstitutions[sub.Type()] = sub
}

// GetSubstitution returns a substitution by its type
func GetSubstitution(subType string) Handler {
	return registeredSubstitutions[subType]
}
