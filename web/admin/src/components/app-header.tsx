import { useNavigate } from "react-router-dom"
import { LogOut, ChevronUp } from "lucide-react"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useAuth } from "@/lib/auth"
import { toast } from "sonner"

interface AppHeaderProps {
  title: string
}

export function AppHeader({ title }: AppHeaderProps) {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    try {
      await logout()
      navigate("/login", { replace: true })
    } catch {
      toast.error("Gagal logout")
    }
  }

  const initials = (user?.name ?? "?")
    .split(" ")
    .map((part) => part[0])
    .slice(0, 2)
    .join("")
    .toUpperCase()

  return (
    <header className="flex h-[var(--header-height)] shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-[var(--header-height)]">
      <div className="flex w-full items-center gap-1 px-4 py-3 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4" />
        <h1 className="text-sm font-semibold">{title}</h1>
        
        <div className="ml-auto flex items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex items-center gap-2 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-ring">
                <Avatar className="size-8">
                  <AvatarFallback className="text-xs">{initials}</AvatarFallback>
                </Avatar>
                <div className="hidden flex-col items-start leading-tight sm:flex">
                  <span className="text-sm font-medium text-foreground">{user?.name}</span>
                  <span className="text-xs text-muted-foreground">{user?.identity_number}</span>
                </div>
                <ChevronUp className="size-4 text-muted-foreground" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleLogout}>
                <LogOut className="mr-2 size-4" />
                Keluar
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  )
}

