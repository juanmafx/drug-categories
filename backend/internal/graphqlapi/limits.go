package graphqlapi

import (
	"errors"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/lexer"
	"github.com/graphql-go/graphql/language/source"
)

const (
	maxQueryDepth     = 5
	maxSelectionNodes = 100
	maxMutationFields = 1
	maxQueryTokens    = 1000
	maxReportedErrors = 10
)

var (
	errUnparsable        = errors.New("query could not be parsed")
	errTooManyTokens     = errors.New("query exceeds the maximum token count")
	errInvalidVariable   = errors.New("string variables must be strings")
	errTooDeep           = errors.New("query exceeds the maximum depth")
	errTooComplex        = errors.New("query exceeds the maximum complexity")
	errTooManyMutations  = errors.New("only one mutation field is allowed per request")
	errUnknownFragment   = errors.New("query references an unknown fragment")
	errDuplicateFragment = errors.New("query defines a fragment more than once")
	errFragmentCycle     = errors.New("query contains a fragment cycle")
	errUnusedFragment    = errors.New("query defines an unused fragment")
)

type limitWalker struct {
	fragments  map[string]*ast.FragmentDefinition
	active     map[string]bool
	reached    map[string]bool
	nodes      int
	rootFields int
}

func enforceTokenLimit(body *source.Source) error {
	next := lexer.Lex(body)
	for count := 0; ; count++ {
		token, err := next(0)
		if err != nil {
			return errUnparsable
		}
		if token.Kind == lexer.EOF {
			return nil
		}
		if count >= maxQueryTokens {
			return errTooManyTokens
		}
	}
}

func enforceStringVariables(operation *ast.OperationDefinition, variables map[string]any) error {
	for _, definition := range operation.VariableDefinitions {
		declared := definition.Type
		if nonNull, ok := declared.(*ast.NonNull); ok {
			declared = nonNull.Type
		}
		named, ok := declared.(*ast.Named)
		if !ok || named.Name == nil || named.Name.Value != "String" {
			continue
		}
		value, present := variables[definition.Variable.Name.Value]
		if !present || value == nil {
			continue
		}
		if _, isString := value.(string); !isString {
			return errInvalidVariable
		}
	}
	return nil
}

func enforceLimits(operation *ast.OperationDefinition, document *ast.Document) error {
	walker := limitWalker{
		fragments: map[string]*ast.FragmentDefinition{},
		active:    map[string]bool{},
		reached:   map[string]bool{},
	}
	for _, definition := range document.Definitions {
		if fragment, ok := definition.(*ast.FragmentDefinition); ok && fragment.Name != nil {
			if _, duplicate := walker.fragments[fragment.Name.Value]; duplicate {
				return errDuplicateFragment
			}
			walker.fragments[fragment.Name.Value] = fragment
		}
	}
	if err := walker.walk(operation.SelectionSet, 1); err != nil {
		return err
	}
	if len(walker.reached) != len(walker.fragments) {
		return errUnusedFragment
	}
	if operation.Operation == ast.OperationTypeMutation && walker.rootFields > maxMutationFields {
		return errTooManyMutations
	}
	return nil
}

func (w *limitWalker) walk(selectionSet *ast.SelectionSet, depth int) error {
	if selectionSet == nil {
		return nil
	}
	if depth > maxQueryDepth {
		return errTooDeep
	}
	for _, selection := range selectionSet.Selections {
		w.nodes++
		if w.nodes > maxSelectionNodes {
			return errTooComplex
		}
		var err error
		switch node := selection.(type) {
		case *ast.Field:
			if depth == 1 {
				w.rootFields++
			}
			err = w.walk(node.SelectionSet, depth+1)
		case *ast.InlineFragment:
			err = w.walk(node.SelectionSet, depth)
		case *ast.FragmentSpread:
			name := node.Name.Value
			fragment, known := w.fragments[name]
			if !known {
				return errUnknownFragment
			}
			if w.active[name] {
				return errFragmentCycle
			}
			w.active[name] = true
			w.reached[name] = true
			err = w.walk(fragment.SelectionSet, depth)
			delete(w.active, name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
