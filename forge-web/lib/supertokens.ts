import ThirdParty, { Github, Google } from "supertokens-auth-react/recipe/thirdparty";
import Session from "supertokens-auth-react/recipe/session";

export const apiDomain =
  process.env.NEXT_PUBLIC_PLATFORM_API_URL ?? "http://localhost:8080";
export const websiteDomain =
  process.env.NEXT_PUBLIC_WEBSITE_DOMAIN ?? "http://localhost:3000";

export const superTokensConfig = {
  appInfo: {
    appName: "Forge",
    apiDomain,
    websiteDomain,
    apiBasePath: "/auth",
    websiteBasePath: "/auth",
  },
  recipeList: [
    ThirdParty.init({
      signInAndUpFeature: { providers: [Github.init(), Google.init()] },
    }),
    Session.init(),
  ],
};
