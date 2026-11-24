package graphqlapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"

	"github.com/juanmafx/drug-categories/backend/internal/domain"
	"github.com/juanmafx/drug-categories/backend/internal/store"
)

const internalErrorMessage = "internal server error"

var (
	ErrRejected     = errors.New("request rejected")
	ErrUnauthorized = errors.New("a valid write token is required")
)

type Request struct {
	Query         string
	OperationName string
	Variables     map[string]any
	CanWrite      bool
}

type Error struct {
	Message string `json:"message"`
}

type Response struct {
	Data   any     `json:"data,omitempty"`
	Errors []Error `json:"errors,omitempty"`
}

type Service struct {
	schema graphql.Schema
	logger *slog.Logger
}

func NewService(repository store.Repository, logger *slog.Logger) (*Service, error) {
	schema, err := buildSchema(repository)
	if err != nil {
		return nil, fmt.Errorf("build schema: %w", err)
	}
	return &Service{schema: schema, logger: logger}, nil
}

func (s *Service) Execute(ctx context.Context, request Request) (Response, error) {
	document, err := parser.Parse(parser.ParseParams{
		Source: source.NewSource(&source.Source{Body: []byte(request.Query), Name: "request"}),
	})
	if err != nil {
		return rejected("query could not be parsed"), ErrRejected
	}

	operation, err := singleOperation(document)
	if err != nil {
		return rejected(err.Error()), ErrRejected
	}
	if err := enforceLimits(operation, document); err != nil {
		return rejected(err.Error()), ErrRejected
	}

	validation := graphql.ValidateDocument(&s.schema, document, nil)
	if !validation.IsValid {
		response := Response{}
		for _, validationError := range validation.Errors {
			response.Errors = append(response.Errors, Error{Message: validationError.Message})
		}
		return response, ErrRejected
	}

	if operation.Operation == ast.OperationTypeMutation && !request.CanWrite {
		return rejected(ErrUnauthorized.Error()), ErrUnauthorized
	}

	result := graphql.Execute(graphql.ExecuteParams{
		Schema:        s.schema,
		AST:           document,
		Args:          request.Variables,
		OperationName: request.OperationName,
		Context:       ctx,
	})
	response := Response{Data: result.Data}
	for _, formatted := range result.Errors {
		response.Errors = append(response.Errors, Error{Message: s.publicMessage(formatted)})
	}
	return response, nil
}

func singleOperation(document *ast.Document) (*ast.OperationDefinition, error) {
	var operation *ast.OperationDefinition
	for _, definition := range document.Definitions {
		candidate, ok := definition.(*ast.OperationDefinition)
		if !ok {
			continue
		}
		if operation != nil {
			return nil, errors.New("only one operation is allowed per request")
		}
		operation = candidate
	}
	if operation == nil {
		return nil, errors.New("query must contain an operation")
	}
	return operation, nil
}

func (s *Service) publicMessage(formatted gqlerrors.FormattedError) string {
	original := rootCause(formatted.OriginalError())
	if original == nil {
		return formatted.Message
	}
	if errors.Is(original, domain.ErrInvalidCategory) || errors.Is(original, domain.ErrDuplicateCategory) {
		return original.Error()
	}
	s.logger.Error("graphql execution failed", "error", original)
	return internalErrorMessage
}

func rootCause(err error) error {
	for err != nil {
		located, ok := err.(*gqlerrors.Error)
		if !ok {
			return err
		}
		if located.OriginalError == nil {
			return nil
		}
		err = located.OriginalError
	}
	return nil
}

func rejected(message string) Response {
	return Response{Errors: []Error{{Message: message}}}
}

func buildSchema(repository store.Repository) (graphql.Schema, error) {
	categoryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Category",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"name":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"description": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"categories": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(categoryType))),
				Resolve: func(params graphql.ResolveParams) (any, error) {
					return repository.List(params.Context)
				},
			},
		},
	})

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createCategory": &graphql.Field{
				Type: graphql.NewNonNull(categoryType),
				Args: graphql.FieldConfigArgument{
					"name":        &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"description": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					name, _ := params.Args["name"].(string)
					description, _ := params.Args["description"].(string)
					input, err := domain.ParseNewCategory(name, description)
					if err != nil {
						return nil, err
					}
					return repository.Create(params.Context, input)
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{Query: queryType, Mutation: mutationType})
}
