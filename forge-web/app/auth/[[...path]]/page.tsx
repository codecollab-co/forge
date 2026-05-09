"use client";

import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { redirectToAuth } from "supertokens-auth-react";
import { canHandleRoute, getRoutingComponent } from "supertokens-auth-react/ui";
import { ThirdPartyPreBuiltUI } from "supertokens-auth-react/recipe/thirdparty/prebuiltui";

// SuperTokens prebuilt UI catch-all for /auth/*.
// Handles the sign-in screen and the OAuth callback for GitHub + Google.
export default function AuthCatchAll() {
  const [routed, setRouted] = useState<ReactElement | null>(null);

  useEffect(() => {
    if (canHandleRoute([ThirdPartyPreBuiltUI])) {
      setRouted(getRoutingComponent([ThirdPartyPreBuiltUI]));
    } else {
      redirectToAuth();
    }
  }, []);

  return routed;
}
