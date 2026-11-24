package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/juanmafx/drug-categories/backend/internal/domain"
	"github.com/juanmafx/drug-categories/backend/internal/store"
)

const createMutation = `mutation($name:String!,$description:String){
  createCategory(name:$name,description:$description){ id name description }
}`

type failingRepository struct{}

func (failingRepository) List(context.Context) ([]domain.Category, error) {
	return nil, errors.New("secret connection detail")
}

func (failingRepository) Create(context.Context, domain.NewCategory) (domain.Category, error) {
	return domain.Category{}, errors.New("secret connection detail")
}

func newService(t *testing.T, repository store.Repository, output io.Writer) *Service {
	t.Helper()
	service, err := NewService(repository, slog.New(slog.NewTextHandler(output, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func defaultService(t *testing.T) *Service {
	t.Helper()
	return newService(t, store.NewMemory(store.DefaultSeed()...), io.Discard)
}

func encode(t *testing.T, response Response) string {
	t.Helper()
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestQueryListsCategories(t *testing.T) {
	response, err := defaultService(t).Execute(context.Background(), Request{Query: `query { categories { id name description } }`})
	if err != nil {
		t.Fatal(err)
	}
	if got := encode(t, response); !strings.Contains(got, `"name":"Analgesics"`) {
		t.Fatalf("got %s", got)
	}
}

func TestMutationCreatesCategory(t *testing.T) {
	service := defaultService(t)
	response, err := service.Execute(context.Background(), Request{
		Query:     createMutation,
		Variables: map[string]any{"name": " Antivirals ", "description": "Treat viral infections"},
		CanWrite:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := encode(t, response); !strings.Contains(got, `"id":2`) || !strings.Contains(got, `"name":"Antivirals"`) {
		t.Fatalf("got %s", got)
	}
}

func TestMutationWithoutDescription(t *testing.T) {
	response, err := defaultService(t).Execute(context.Background(), Request{
		Query:     createMutation,
		Variables: map[string]any{"name": "Antivirals"},
		CanWrite:  true,
	})
	if err != nil || len(response.Errors) != 0 {
		t.Fatalf("err=%v errors=%v", err, response.Errors)
	}
}

func TestMutationRequiresWriteAccess(t *testing.T) {
	service := defaultService(t)
	_, err := service.Execute(context.Background(), Request{
		Query:     createMutation,
		Variables: map[string]any{"name": "Blocked"},
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
	response, _ := service.Execute(context.Background(), Request{Query: `{ categories { name } }`})
	if strings.Contains(encode(t, response), "Blocked") {
		t.Fatal("unauthorized mutation was applied")
	}
}

func TestQueryMentioningMutationNameDoesNotMutate(t *testing.T) {
	service := defaultService(t)
	response, err := service.Execute(context.Background(), Request{Query: `query createCategory { categories { name } }`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(encode(t, response), `"name"`) != 1 {
		t.Fatalf("got %s", encode(t, response))
	}
}

func TestDomainErrorsArePublic(t *testing.T) {
	service := defaultService(t)
	response, err := service.Execute(context.Background(), Request{
		Query:     createMutation,
		Variables: map[string]any{"name": "   "},
		CanWrite:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Errors) != 1 || !strings.Contains(response.Errors[0].Message, "name is required") {
		t.Fatalf("got %+v", response.Errors)
	}

	response, err = service.Execute(context.Background(), Request{
		Query:     createMutation,
		Variables: map[string]any{"name": "analgesics"},
		CanWrite:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Errors) != 1 || !strings.Contains(response.Errors[0].Message, "already exists") {
		t.Fatalf("got %+v", response.Errors)
	}
}

func TestInternalErrorsAreMaskedAndLogged(t *testing.T) {
	var logs bytes.Buffer
	service := newService(t, failingRepository{}, &logs)
	response, err := service.Execute(context.Background(), Request{Query: `{ categories { id } }`})
	if err != nil {
		t.Fatal(err)
	}
	encoded := encode(t, response)
	if strings.Contains(encoded, "secret") || !strings.Contains(encoded, internalErrorMessage) {
		t.Fatalf("got %s", encoded)
	}
	if !strings.Contains(logs.String(), "secret connection detail") {
		t.Fatalf("expected server-side log, got %q", logs.String())
	}
}

func TestRejectedDocuments(t *testing.T) {
	deep := `{ categories { a { b { c { d { e } } } } } }`
	wide := "{ " + strings.Repeat("categories { id } ", 60) + "}"
	aliases := "mutation { "
	for index := range 2 {
		aliases += fmt.Sprintf(`a%d: createCategory(name:"N%d"){ id } `, index, index)
	}
	aliases += "}"
	var spreads strings.Builder
	spreads.WriteString("{ ...F0 } ")
	for index := range 12 {
		fmt.Fprintf(&spreads, "fragment F%d on Query { ...F%d ...F%d } ", index, index+1, index+1)
	}
	spreads.WriteString("fragment F12 on Query { categories { id } }")

	cases := map[string]string{
		"syntax error":        `{ categories `,
		"unknown field":       `{ unknownField }`,
		"too deep":            deep,
		"too wide":            wide,
		"mutation aliases":    aliases,
		"fragment cycle":      `{ ...A } fragment A on Query { ...B } fragment B on Query { ...A }`,
		"fragment fan-out":    spreads.String(),
		"unknown fragment":    `{ ...Missing }`,
		"multiple operations": `query A { categories { id } } query B { categories { id } }`,
		"fragment only":       `fragment A on Query { categories { id } }`,
	}
	service := defaultService(t)
	for label, query := range cases {
		t.Run(label, func(t *testing.T) {
			response, err := service.Execute(context.Background(), Request{Query: query, CanWrite: true})
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("got %v", err)
			}
			if len(response.Errors) == 0 {
				t.Fatal("expected error message")
			}
		})
	}

	listed, _ := service.Execute(context.Background(), Request{Query: `{ categories { name } }`})
	if strings.Contains(encode(t, listed), "N0") {
		t.Fatal("rejected mutation was applied")
	}
}

func TestMissingRequiredVariableIsReported(t *testing.T) {
	response, err := defaultService(t).Execute(context.Background(), Request{Query: createMutation, CanWrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Errors) == 0 {
		t.Fatal("expected variable error")
	}
}
