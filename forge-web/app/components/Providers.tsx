"use client";

import { useEffect, useState } from "react";
import SuperTokens, { SuperTokensWrapper } from "supertokens-auth-react";
import { superTokensConfig } from "@/lib/supertokens";

if (typeof window !== "undefined") {
  SuperTokens.init(superTokensConfig);
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false);
  useEffect(() => setReady(true), []);
  if (!ready) return null;
  return <SuperTokensWrapper>{children}</SuperTokensWrapper>;
}
