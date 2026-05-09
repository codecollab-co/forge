import "./globals.css";
import type { Metadata, Viewport } from "next";
import { Providers } from "./components/Providers";
import { TopNav } from "./components/TopNav";

export const metadata: Metadata = {
  title: "Forge",
  description: "AI-native Git host",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-white text-zinc-900 antialiased dark:bg-zinc-950 dark:text-zinc-100">
        <Providers>
          <TopNav />
          {children}
        </Providers>
      </body>
    </html>
  );
}
