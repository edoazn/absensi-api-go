import type { ReactNode } from "react"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { AppHeader } from "@/components/app-header"

export function AppLayout({ title, children }: { title: string; children: ReactNode }) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <AppHeader title={title} />
        <div className="flex flex-1 flex-col">
          <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 lg:px-6">
            <main className="flex-1 space-y-4 px-4 lg:px-0">
              {children}
            </main>
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}

