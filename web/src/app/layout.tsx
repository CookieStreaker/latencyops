import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Providers } from "@/components/providers";
import { Activity, ShieldAlert, Settings, Bell, Network } from "lucide-react";
import Link from "next/link";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });

export const metadata: Metadata = {
  title: "LatencyOps | API Monitor",
  description: "Real-time API health and rate-limit alerting.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${inter.variable} font-sans min-h-screen flex flex-col md:flex-row bg-zinc-950`}>
        <Providers>
          {/* Desktop Sidebar */}
          <aside className="w-64 border-r border-zinc-800/80 bg-zinc-950 hidden md:flex flex-col h-screen sticky top-0">
            <div className="flex-1 px-4 py-6 space-y-8">
              
              {/* Brand */}
              <div className="flex items-center space-x-2.5 px-2">
                <div className="h-7 w-7 rounded-md bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center">
                  <Activity className="h-4 w-4 text-emerald-400" />
                </div>
                <span className="font-semibold tracking-tight text-zinc-100">
                  Latency<span className="text-zinc-500">Ops</span>
                </span>
              </div>

              {/* Navigation */}
              <nav className="space-y-1">
                <NavItem href="/" icon={Network} label="Endpoints" active />
                <NavItem href="/alerts" icon={Bell} label="Alert Rules" />
                <NavItem href="/security" icon={ShieldAlert} label="SSRF Audit" />
                <NavItem href="/settings" icon={Settings} label="Settings" />
              </nav>
            </div>

            {/* Workspace Context */}
            <div className="p-4 border-t border-zinc-800/80">
              <div className="px-2">
                <p className="text-[10px] font-mono font-medium text-zinc-500 uppercase tracking-wider mb-1">
                  Active Workspace
                </p>
                <p className="text-xs font-mono text-zinc-300 truncate">
                  ws_prod_default
                </p>
              </div>
            </div>
          </aside>

          {/* Main Content Pane */}
          <main className="flex-1 flex flex-col h-screen overflow-y-auto bg-zinc-950">
            {children}
          </main>
        </Providers>
      </body>
    </html>
  );
}

// Minimal Navigation Item Component
function NavItem({ href, icon: Icon, label, active }: { href: string; icon: any; label: string; active?: boolean }) {
  return (
    <Link
      href={href}
      className={`flex items-center space-x-3 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
        active 
          ? "bg-zinc-900/80 text-zinc-100 border border-zinc-800" 
          : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/50 border border-transparent"
      }`}
    >
      <Icon className="h-4 w-4 shrink-0" />
      <span>{label}</span>
    </Link>
  );
}