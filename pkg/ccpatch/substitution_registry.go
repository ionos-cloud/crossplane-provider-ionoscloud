package ccpatch

type SubstitutionHandler interface {
	Type() string
	WriteState(identifier Identifier, state *GlobalState, sub Substitution) error
}

var registeredSubstitutions = make(map[string]SubstitutionHandler)

// RegisterSubstitution registers a new substitution
func RegisterSubstitution(sub SubstitutionHandler) {
	if sub == nil {
		panic("cannot register a nil substitution")
	}

	if sub.Type() == "" {
		panic("cannot register a substitution with an empty type")
	}

	registeredSubstitutions[sub.Type()] = sub
}

// GetSubstitution returns a substitution by its type
func GetSubstitution(subType string) SubstitutionHandler {
	return registeredSubstitutions[subType]
}
