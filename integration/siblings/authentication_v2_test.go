package siblings_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authentication "github.com/faustbrian/go-authentication/v2"
	authenticationhttp "github.com/faustbrian/go-authentication/v2/adapters/http"
	"github.com/faustbrian/go-authentication/v2/bearer"
	authorization "github.com/faustbrian/go-authorization"
	authorizationhttp "github.com/faustbrian/go-authorization/adapters/http"
	"github.com/faustbrian/go-authorization/authn"
	"github.com/faustbrian/go-service/serverhttp"
)

func TestAuthenticationV2ServiceComposition(t *testing.T) {
	for _, test := range []struct {
		name   string
		header string
		status int
	}{
		{name: "accepted", header: "Bearer fixture-token", status: http.StatusNoContent},
		{name: "missing", status: http.StatusUnauthorized},
		{name: "rejected", header: "Bearer other-fixture", status: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			extractor, err := authenticationhttp.NewExtractor(authenticationhttp.BearerAuthorization())
			if err != nil {
				t.Fatal("extractor setup failed")
			}
			authenticator, err := bearer.New(bearer.ValidatorFunc(func(_ context.Context, token string) (authentication.Principal, error) {
				if token != "fixture-token" {
					return authentication.Principal{}, authentication.NewFailure(authentication.FailureRejected)
				}
				return authentication.NewPrincipalWithOptions(authentication.PrincipalSpec{
					Subject: "viewer", Method: "bearer",
				}, authentication.WithMaxPrincipalStringBytes(6))
			}))
			if err != nil {
				t.Fatal("authenticator setup failed")
			}
			challenge, err := authentication.NewChallenge("Bearer", nil)
			if err != nil {
				t.Fatal("challenge setup failed")
			}
			authenticate, err := authenticationhttp.NewMiddleware(extractor, authenticator,
				authenticationhttp.WithChallenges(challenge))
			if err != nil {
				t.Fatal("middleware setup failed")
			}
			var mapped, served bool
			application := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				served = true
				principal, ok := authentication.PrincipalFromContext(request.Context())
				if !ok || principal.Subject() != "viewer" || principal.Method() != "bearer" {
					t.Error("application lost authenticated principal")
				}
				if decision, ok := authorizationhttp.DecisionFromContext(request.Context()); !ok || decision.Outcome != authorization.Allow {
					t.Error("application lost allowed decision")
				}
				writer.WriteHeader(http.StatusNoContent)
			})
			authorize, err := authorizationhttp.NewHandler(allowAuthorizer{}, func(request *http.Request) (authorization.Request, error) {
				mapped = true
				principal, ok := authentication.PrincipalFromContext(request.Context())
				if !ok {
					t.Error("mapper lost authenticated principal")
				}
				subject, mapErr := authn.Subject(principal, authn.Config{Kind: authorization.SubjectServiceAccount})
				if mapErr != nil {
					return authorization.Request{}, mapErr
				}
				if subject.ID != "viewer" || subject.Kind != authorization.SubjectServiceAccount {
					t.Error("mapper changed authenticated identity")
				}
				return authorization.Request{Subject: subject, Action: "article.read",
					Resource: authorization.Resource{Type: "article", ID: "article-1"}}, nil
			}, application)
			if err != nil {
				t.Fatal("authorization setup failed")
			}
			request := httptest.NewRequest(http.MethodGet, "/articles/article-1", nil)
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}
			response := httptest.NewRecorder()
			handler, err := serverhttp.Chain(authorize, serverhttp.Recover(), authenticate)
			if err != nil {
				t.Fatal("service chain setup failed")
			}
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			allowed := test.status == http.StatusNoContent
			if mapped != allowed || served != allowed {
				t.Fatal("unexpected downstream execution")
			}
			if !allowed && response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatal("missing bearer challenge")
			}
		})
	}
}

func TestAuthorizationV1HTTPAdoptionFailsClosed(t *testing.T) {
	extractor, err := authenticationhttp.NewExtractor(authenticationhttp.BearerAuthorization())
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := bearer.New(bearer.ValidatorFunc(func(_ context.Context, token string) (authentication.Principal, error) {
		if token != "fixture-token" {
			return authentication.Principal{}, authentication.NewFailure(authentication.FailureRejected)
		}
		return authentication.NewPrincipal(authentication.PrincipalSpec{Subject: "viewer", Method: "bearer"})
	}))
	if err != nil {
		t.Fatal(err)
	}
	authenticate, err := authenticationhttp.NewMiddleware(extractor, authenticator)
	if err != nil {
		t.Fatal(err)
	}
	mappingFailure := errors.New("fixture mapping failure")
	evaluationFailure := errors.New("fixture evaluation failure")
	for _, test := range []struct {
		name          string
		outcome       authorization.Outcome
		mappingErr    error
		evaluationErr error
		custom        bool
		status        int
	}{
		{name: "allow", outcome: authorization.Allow, status: http.StatusOK},
		{name: "deny", outcome: authorization.Deny, status: http.StatusForbidden},
		{name: "not applicable", outcome: authorization.NotApplicable, status: http.StatusForbidden},
		{name: "mapping failure", mappingErr: mappingFailure, status: http.StatusInternalServerError},
		{name: "evaluation failure overrides allow", outcome: authorization.Allow, evaluationErr: evaluationFailure, status: http.StatusInternalServerError},
		{name: "invalid outcome", outcome: authorization.Outcome(99), status: http.StatusInternalServerError},
		{name: "custom denial", outcome: authorization.Deny, custom: true, status: http.StatusTeapot},
		{name: "custom error", outcome: authorization.Allow, evaluationErr: evaluationFailure, custom: true, status: http.StatusBadGateway},
	} {
		t.Run(test.name, func(t *testing.T) {
			decision := authorization.Decision{Outcome: test.outcome, Reason: "fixture-reason", Revision: 7}
			var evaluated, served int
			authorizer := authorizationDecisionFunc(func(ctx context.Context, request authorization.Request) (authorization.Decision, error) {
				evaluated++
				principal, ok := authentication.PrincipalFromContext(ctx)
				if !ok || principal.Subject() != "viewer" || principal.Method() != "bearer" {
					t.Error("authorizer lost authenticated context")
				}
				if request.Subject.ID != "viewer" || request.Subject.Kind != authorization.SubjectServiceAccount ||
					request.Action != "article.read" || request.Resource.Type != "article" || request.Resource.ID != "article-1" {
					t.Errorf("authorization request = %#v", request)
				}
				return decision, test.evaluationErr
			})
			mapper := func(request *http.Request) (authorization.Request, error) {
				principal, ok := authentication.PrincipalFromContext(request.Context())
				if !ok {
					t.Error("mapper lost authenticated principal")
				}
				if test.mappingErr != nil {
					return authorization.Request{}, test.mappingErr
				}
				subject, err := authn.Subject(principal, authn.Config{Kind: authorization.SubjectServiceAccount})
				if err != nil {
					return authorization.Request{}, err
				}
				return authorization.Request{Subject: subject, Action: "article.read", Resource: authorization.Resource{Type: "article", ID: "article-1"}}, nil
			}
			application := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				served++
				got, ok := authorizationhttp.DecisionFromContext(request.Context())
				if !ok || got.Outcome != authorization.Allow || got.Reason != "fixture-reason" || got.Revision != 7 {
					t.Errorf("application decision = %#v, present = %v", got, ok)
				}
				writer.Header().Set("X-Protected", "reached")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte("protected-article"))
			})
			var options []authorizationhttp.Option
			if test.custom {
				options = append(options, authorizationhttp.WithDeniedHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					got, ok := authorizationhttp.DecisionFromContext(request.Context())
					if !ok || got.Outcome != authorization.Deny || got.Reason != "fixture-reason" || got.Revision != 7 {
						t.Errorf("denial decision = %#v, present = %v", got, ok)
					}
					writer.WriteHeader(http.StatusTeapot)
				})), authorizationhttp.WithErrorHandler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					got, ok := authorizationhttp.ErrorFromContext(request.Context())
					if !ok || !errors.Is(got, evaluationFailure) {
						t.Errorf("handler error = %v, present = %v", got, ok)
					}
					writer.WriteHeader(http.StatusBadGateway)
				})))
			}
			authorize, err := authorizationhttp.NewHandler(authorizer, mapper, application, options...)
			if err != nil {
				t.Fatal(err)
			}
			handler, err := serverhttp.Chain(authorize, serverhttp.Recover(), authenticate)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/articles/article-1", nil)
			request.Header.Set("Authorization", "Bearer fixture-token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			wantEvaluated := 1
			if test.mappingErr != nil {
				wantEvaluated = 0
			}
			if evaluated != wantEvaluated {
				t.Errorf("evaluations = %d, want %d", evaluated, wantEvaluated)
			}
			if test.name == "allow" {
				if served != 1 || response.Body.String() != "protected-article" || response.Header().Get("X-Protected") != "reached" {
					t.Fatal("allowed request lost protected response")
				}
			} else if served != 0 || response.Body.Len() != 0 || response.Header().Get("X-Protected") != "" {
				t.Fatal("refused request reached protected application")
			}
		})
	}
}

type authorizationDecisionFunc func(context.Context, authorization.Request) (authorization.Decision, error)

func (decide authorizationDecisionFunc) Decide(ctx context.Context, request authorization.Request) (authorization.Decision, error) {
	return decide(ctx, request)
}
