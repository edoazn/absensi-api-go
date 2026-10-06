import { useCallback, useEffect, useState } from "react"
import { api, apiBlob } from "@/lib/api"
import { AppLayout } from "@/components/app-layout"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Download, RefreshCw } from "lucide-react"
import { toast } from "sonner"

interface ReportItem {
  id: number
  student: string
  identity: string
  status: string
  method: string
  distance: number | null
  created_at: string
  course?: string
  course_code?: string
  location?: string
}

interface ReportMeta {
  current_page: number
  per_page: number
  total: number
  last_page: number
}

export function AttendancesPage() {
  const [items, setItems] = useState<ReportItem[]>([])
  const [meta, setMeta] = useState<ReportMeta>({ current_page: 1, per_page: 15, total: 0, last_page: 1 })
  const [schedules, setSchedules] = useState<{ id: number; label: string }[]>([])
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [startDate, setStartDate] = useState("")
  const [endDate, setEndDate] = useState("")
  const [scheduleId, setScheduleId] = useState("all")
  const [page, setPage] = useState(1)

  const loadSchedules = useCallback(async () => {
    try {
      const env = await api<
        { id: number; course: string; course_code: string; class: string; start_time: string }[]
      >("/schedules")
      setSchedules(
        (env.data ?? []).map((s) => ({
          id: s.id,
          label: `${s.course_code} - ${s.class} (${s.start_time.slice(0, 10)})`,
        })),
      )
    } catch {
      // filter jadwal opsional
    }
  }, [])

  const loadReport = useCallback(
    async (targetPage = 1) => {
      setLoading(true)
      try {
        const params = new URLSearchParams()
        if (startDate) params.set("start_date", startDate)
        if (endDate) params.set("end_date", endDate)
        if (scheduleId !== "all") params.set("schedule_id", scheduleId)
        params.set("page", String(targetPage))
        const env = await api<{ items: ReportItem[]; meta: ReportMeta }>(
          `/reports/attendance?${params}`,
        )
        setItems(env.data?.items ?? [])
        if (env.data?.meta) setMeta(env.data.meta)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Gagal memuat laporan")
      } finally {
        setLoading(false)
      }
    },
    [startDate, endDate, scheduleId],
  )

  useEffect(() => {
    loadSchedules()
  }, [loadSchedules])
  useEffect(() => {
    setPage(1)
    loadReport(1)
  }, [loadReport])

  const gotoPage = (next: number) => {
    if (next < 1 || next > meta.last_page) return
    setPage(next)
    loadReport(next)
  }

  const handleExport = async () => {
    setExporting(true)
    try {
      const params = new URLSearchParams()
      if (startDate) params.set("start_date", startDate)
      if (endDate) params.set("end_date", endDate)
      if (scheduleId !== "all") params.set("schedule_id", scheduleId)
      const blob = await apiBlob(`/reports/attendance/export?${params}`)
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `laporan-absensi-${new Date().toISOString().slice(0, 10)}.xlsx`
      a.click()
      URL.revokeObjectURL(url)
      toast.success("Excel berhasil diunduh")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal mengunduh")
    } finally {
      setExporting(false)
    }
  }

  return (
    <AppLayout title="Absensi">
      <div className="flex flex-wrap items-end gap-3">
        <div className="space-y-1.5">
          <Label htmlFor="f-start">Dari</Label>
          <Input
            id="f-start"
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="f-end">Sampai</Label>
          <Input
            id="f-end"
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label>Jadwal</Label>
          <Select value={scheduleId} onValueChange={setScheduleId}>
            <SelectTrigger className="w-64">
              <SelectValue placeholder="Semua jadwal" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua jadwal</SelectItem>
              {schedules.map((s) => (
                <SelectItem key={s.id} value={String(s.id)}>
                  {s.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button variant="outline" onClick={() => loadReport(1)}>
          <RefreshCw />
          Refresh
        </Button>
        <Button className="ml-auto" onClick={handleExport} disabled={exporting}>
          {exporting ? <Spinner className="size-4" /> : <Download />}
          Export Excel
        </Button>
      </div>

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {meta.total} baris — halaman {meta.current_page} / {meta.last_page}
        </p>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1 || loading}
            onClick={() => gotoPage(page - 1)}
          >
            Sebelumnya
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= meta.last_page || loading}
            onClick={() => gotoPage(page + 1)}
          >
            Berikutnya
          </Button>
        </div>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Mahasiswa</TableHead>
              <TableHead>NIM</TableHead>
              <TableHead>Mata Kuliah</TableHead>
              <TableHead>Lokasi</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Metode</TableHead>
              <TableHead>Jarak</TableHead>
              <TableHead>Waktu</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && (
              <TableRow>
                <TableCell colSpan={8} className="py-8 text-center">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-medium">{row.student}</TableCell>
                  <TableCell className="font-mono text-xs">{row.identity}</TableCell>
                  <TableCell>
                    <p>{row.course ?? "-"}</p>
                    <p className="font-mono text-xs text-muted-foreground">{row.course_code}</p>
                  </TableCell>
                  <TableCell>{row.location ?? "-"}</TableCell>
                  <TableCell>
                    {row.status === "hadir" ? (
                      <Badge className="bg-emerald-600 hover:bg-emerald-600">Hadir</Badge>
                    ) : (
                      <Badge variant="destructive">Ditolak</Badge>
                    )}
                  </TableCell>
                  <TableCell>{row.method}</TableCell>
                  <TableCell>{row.distance != null ? `${row.distance} m` : "-"}</TableCell>
                  <TableCell className="whitespace-nowrap text-muted-foreground">
                    {row.created_at}
                  </TableCell>
                </TableRow>
              ))}
            {!loading && items.length === 0 && (
              <TableRow>
                <TableCell colSpan={8} className="text-center text-muted-foreground">
                  Tidak ada data absensi.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </AppLayout>
  )
}
