import { NavLink } from "react-router-dom"
import {
  CalendarClock,
  ClipboardCheck,
  LayoutDashboard,
  MapPin,
  BookOpen,
  Users,
  UserSquare2,
} from "lucide-react"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,

  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

const items = [
  { title: "Dashboard", to: "/", icon: LayoutDashboard },
  { title: "Pengguna", to: "/users", icon: Users },
  { title: "Kelas", to: "/classes", icon: UserSquare2 },
  { title: "Mata Kuliah", to: "/courses", icon: BookOpen },
  { title: "Lokasi", to: "/locations", icon: MapPin },
  { title: "Jadwal", to: "/schedules", icon: CalendarClock },
  { title: "Absensi", to: "/attendances", icon: ClipboardCheck },
]

export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <a href="/">
                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                  <ClipboardCheck className="size-4" />
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">Absensi</span>
                  <span className="truncate text-xs text-muted-foreground">Panel Admin</span>
                </div>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Menu Utama</SidebarGroupLabel>
          <SidebarMenu>
            {items.map((item) => (
              <SidebarMenuItem key={item.to}>
                <NavLink to={item.to} end={item.to === "/"}>
                  {({ isActive }) => (
                    <SidebarMenuButton isActive={isActive} tooltip={item.title}>
                      <item.icon className="size-4" />
                      <span>{item.title}</span>
                    </SidebarMenuButton>
                  )}
                </NavLink>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
