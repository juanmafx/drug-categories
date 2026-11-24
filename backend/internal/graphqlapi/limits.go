package graphqlapi

import (
	"errors"

	"github.com/graphql-go/graphql/language/ast"
)

const (
	maxQueryDepth     = 5
	maxSelectionNodes = 100
	maxMutationFields = 1
)

var (
	errTooDeep          = errors.New("query exceeds the maximum depth")
	errTooComplex       = errors.New("query exceeds the maximum complexity")
	errTooManyMutations = errors.New("only one mutation field is allowed per request")
	errUnknownFragment  = errors.New("query references an unknown fragment")
)

type limitWalker struct {
	fragments  map[string]*ast.FragmentDefinition
	nodes      int
	rootFields int
}

func enforceLimits(operation *ast.OperationDefinition, document *ast.Document) error {
	walker := limitWalker{fragments: map[string]*ast.FragmentDefinition{}}
	for _, definition := range document.Definitions {
		if fragment, ok := definition.(*ast.FragmentDefinition); ok && fragment.Name != nil {
			walker.fragments[fragment.Name.Value] = fragment
		}
	}
	if err := walker.walk(operation.SelectionSet, 1); err != nil {
		return err
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
			fragment, known := w.fragments[node.Name.Value]
			if !known {
				return errUnknownFragment
			}
			err = w.walk(fragment.SelectionSet, depth)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
