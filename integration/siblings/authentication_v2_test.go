package siblings_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authentication "github.com/faustbrian/go-authentication/v2"
	authenticationhttp "github.com/faustbrian/go-authentication/v2/adapters/http"
	"github.com/faustbrian/go-authentication/v2/bearer"
	authorization "github.com/faustbrian/go-authorization"
	authorizationhttp "github.com/faustbrian/go-authorization/authhttp"
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
