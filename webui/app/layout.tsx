import type { Metadata } from 'next'
import { Open_Sans } from 'next/font/google'

import { ApiProvider, ThemeProvider } from './providers'
import { Sidebar } from '@/components/layout/Sidebar'
import { Header } from '@/components/layout/Header'
import './globals.css'
import { ConnectionStatus } from '@/components/ConnectionStatus'

const font = Open_Sans({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'Bacalhau',
  description: 'Web UI for Bacalhau',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className={font.className}>
        <ThemeProvider
          attribute="class"
          defaultTheme="light"
          enableSystem
          disableTransitionOnChange
        >
          <ApiProvider>
            <div className="min-h-screen w-full">
              <Sidebar />
              <div className="md:ml-[220px] lg:ml-[280px] flex flex-col min-h-screen">
                <Header />
                <main className="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
                  {children}
                  <ConnectionStatus />
                </main>
              </div>
            </div>
          </ApiProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
