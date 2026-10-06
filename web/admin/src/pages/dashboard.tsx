import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { Envelope } from "@/lib/api"
import { AppLayout } from "@/components/app-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { BookOpen, MapPin, Users } from "lucide-react"

interface DashboardData {
  totals: {
    users: number
    courses: number
    locations: number
    classes: number
    today: number
    hadir: number
    terlambat: number
    ditolak: number
  }
  recent: {
    id: number
    student: string
    course: string
    status: string
    method: string
    created_at: string
  }[]
  schedules: {
    id: number
    course: string
    class: string
    location: string
    start: string
    end: string
    is_active: boolean
  }[]
}

export function DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState("")

  useEffect(() => {
    api<DashboardData>("/admin/dashboard")
      .then((env: Envelope<DashboardData>) => setData(env.data ?? null))
      .catch((err) => setError(err.message))
  }, [])

  const cards = [
    { title: "Pengguna", value: data?.totals.users, icon: Users },
    { title: "Kelas", value: data?.totals.classes, icon: BookOpen },
    { title: "Mata Kuliah", value: data?.totals.courses, icon: BookOpen },
    { title: "Lokasi", value: data?.totals.locations, icon: MapPin },
  ]

  return (
    <AppLayout title="Dashboard">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <Card key={card.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
              <card.icon className="size-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {card.value == null ? (
                <Skeleton className="h-8 w-16" />
              ) : (
                <div className="text-2xl font-bold">{card.value}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Absensi Hari Ini</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.totals.today ?? "-"}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Hadir</CardTitle>
          </CardHeader>
          <CardContent className="flex items-center gap-2">
            <Badge className="bg-emerald-600 hover:bg-emerald-600">Hadir</Badge>
            <span className="text-2xl font-bold">{data?.totals.hadir ?? "-"}</span>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Terlambat</CardTitle>
          </CardHeader>
          <CardContent className="flex items-center gap-2">
            <Badge className="bg-amber-500 hover:bg-amber-500">Terlambat</Badge>
            <span className="text-2xl font-bold">{data?.totals.terlambat ?? "-"}</span>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Ditolak</CardTitle>
          </CardHeader>
          <CardContent className="flex items-center gap-2">
            <Badge variant="destructive">Ditolak</Badge>
            <span className="text-2xl font-bold">{data?.totals.ditolak ?? "-"}</span>
          </CardContent>
        </Card>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Absensi Terbaru</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Mahasiswa</TableHead>
                  <TableHead>Mata Kuliah</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Waktu</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(data?.recent ?? []).map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-medium">{row.student}</TableCell>
                    <TableCell>{row.course}</TableCell>
                    <TableCell>
                      {row.status === "hadir" ? (
                        <Badge className="bg-emerald-600 hover:bg-emerald-600">Hadir</Badge>
                      ) : row.status === "terlambat" ? (
                        <Badge className="bg-amber-500 hover:bg-amber-500">Terlambat</Badge>
                      ) : (
                        <Badge variant="destructive">Ditolak</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{row.created_at}</TableCell>
                  </TableRow>
                ))}
                {data && data.recent.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground">
                      Belum ada absensi.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Jadwal Hari Ini</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {(data?.schedules ?? []).map((s) => (
              <div key={s.id} className="flex items-center justify-between rounded-lg border p-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{s.course}</p>
                  <p className="text-xs text-muted-foreground">
                    {s.class} • {s.location}
                  </p>
                </div>
                <div className="ml-auto flex items-center gap-2">
                  <span className="font-mono text-xs text-muted-foreground">
                    {s.start}–{s.end}
                  </span>
                  {s.is_active ? (
                    <Badge>Berlangsung</Badge>
                  ) : (
                    <Badge variant="secondary">Terjadwal</Badge>
                  )}
                </div>
              </div>
            ))}
            {data && data.schedules.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">
                Tidak ada jadwal hari ini.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </AppLayout>
  )
}
