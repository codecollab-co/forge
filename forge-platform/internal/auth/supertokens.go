// Package auth — SuperTokens initialization for slice 2.
//
// Mounts ThirdParty (GitHub, Google) + Session recipes. The OnSignInUp hook
// upserts platform.users via the supplied callback so user state lives in our
// own Postgres, not just SuperTokens.
package auth

import (
	"context"
	"errors"
	"os"

	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/thirdparty"
	"github.com/supertokens/supertokens-golang/recipe/thirdparty/tpmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
)

type SignInUp struct {
	SuperTokensID string
	Provider      string
	ExternalID    string
	Email         string
	DisplayName   string
	AvatarURL     string
}

type OnSignInUp func(ctx context.Context, e SignInUp) error

func InitSuperTokens(onSignInUp OnSignInUp) error {
	connectionURI := os.Getenv("ST_CONNECTION_URI")
	if connectionURI == "" {
		return errors.New("ST_CONNECTION_URI is empty")
	}
	apiDomain := envOr("API_DOMAIN", "http://localhost:8080")
	websiteDomain := envOr("WEBSITE_DOMAIN", "http://localhost:3000")

	providers := []tpmodels.ProviderInput{}
	if id := os.Getenv("GITHUB_CLIENT_ID"); id != "" {
		providers = append(providers, tpmodels.ProviderInput{
			Config: tpmodels.ProviderConfig{
				ThirdPartyId: "github",
				Clients: []tpmodels.ProviderClientConfig{{
					ClientID:     id,
					ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
				}},
			},
		})
	}
	if id := os.Getenv("GOOGLE_CLIENT_ID"); id != "" {
		providers = append(providers, tpmodels.ProviderInput{
			Config: tpmodels.ProviderConfig{
				ThirdPartyId: "google",
				Clients: []tpmodels.ProviderClientConfig{{
					ClientID:     id,
					ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
				}},
			},
		})
	}

	tpRecipe := thirdparty.Init(&tpmodels.TypeInput{
		SignInAndUpFeature: tpmodels.TypeInputSignInAndUp{Providers: providers},
		Override: &tpmodels.OverrideStruct{
			APIs: func(orig tpmodels.APIInterface) tpmodels.APIInterface {
				inner := *orig.SignInUpPOST
				next := func(
					provider *tpmodels.TypeProvider,
					input tpmodels.TypeSignInUpInput,
					tenantID string,
					options tpmodels.APIOptions,
					userContext supertokens.UserContext,
				) (tpmodels.SignInUpPOSTResponse, error) {
					resp, err := inner(provider, input, tenantID, options, userContext)
					if err != nil || resp.OK == nil {
						return resp, err
					}
					ev := SignInUp{
						SuperTokensID: resp.OK.User.ID,
						Provider:      resp.OK.User.ThirdParty.ID,
						ExternalID:    resp.OK.User.ThirdParty.UserID,
						Email:         resp.OK.User.Email,
					}
					if raw, ok := resp.OK.RawUserInfoFromProvider.FromUserInfoAPI["name"].(string); ok {
						ev.DisplayName = raw
					}
					if raw, ok := resp.OK.RawUserInfoFromProvider.FromUserInfoAPI["avatar_url"].(string); ok {
						ev.AvatarURL = raw
					} else if raw, ok := resp.OK.RawUserInfoFromProvider.FromUserInfoAPI["picture"].(string); ok {
						ev.AvatarURL = raw
					}
					if cbErr := onSignInUp(options.Req.Context(), ev); cbErr != nil {
						return resp, cbErr
					}
					return resp, nil
				}
				orig.SignInUpPOST = &next
				return orig
			},
		},
	})

	return supertokens.Init(supertokens.TypeInput{
		Supertokens: &supertokens.ConnectionInfo{ConnectionURI: connectionURI},
		AppInfo: supertokens.AppInfo{
			AppName:       "Forge",
			APIDomain:     apiDomain,
			WebsiteDomain: websiteDomain,
		},
		RecipeList: []supertokens.Recipe{tpRecipe, session.Init(nil)},
	})
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
