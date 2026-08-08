"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import { useState } from "react";

export function Providers({ children }: { children: React.ReactNode }) {
  // Initialize QueryClient in state to avoid React Suspense boundary resets
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5 * 1000,       // Data is fresh for 5 seconds
            refetchInterval: 10 * 1000, // Background poll every 10 seconds
            retry: 1,                   // Don't spam retries on 404/500s
            refetchOnWindowFocus: true,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <NextThemesProvider 
        attribute="class" 
        defaultTheme="dark" 
        enableSystem={false}
        forcedTheme="dark"
      >
        {children}
      </NextThemesProvider>
    </QueryClientProvider>
  );
}