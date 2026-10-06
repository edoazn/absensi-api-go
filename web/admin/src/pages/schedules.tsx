import { useCallback, useEffect, useState } from "react"
import type { FormEvent } from "react"
import { api } from "@/lib/api"
import { ApiError, apiBlob } from "@/lib/api"
import { AppLayout } from "@/components/app-layout"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
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
import { KeyRound, Plus, QrCode, RefreshCw } from "lucide-react"
import { toast } from "sonner"
import { formatDateTime, toDatetimeLocal } from "@/lib/format"

interface Option {
  id: number
  name: string
}

interface LocationRow {
  id: number
  name: string
}

interface ScheduleRow {
  id: number
  class_id: number
  course_id: number
  location_id: number
  class: string
  course: string
  course_code: string
  location: string
  start_time: string
  end_time: string
  is_recurring: boolean
  repeat_days: number[] | null
  recurrence_end: string | null
  is_active: boolean
  attendance_code: string | null
  code_expires_at: string | null
  has_qr: boolean
}

interface ScheduleFormState {
  class_id: string
  course_id: string
  location_id: string
  start_time: string
  end_time: string
  is_recurring: boolean
  repeat_days: number[]
  recurrence_end: string
}

const DAY_NAMES = ["Min", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"]

const emptyForm: ScheduleFormState = {
  class_id: "",
  course_id: "",
  location_id: "",
  start_time: "",
  end_time: "",
  is_recurring: false,
  repeat_days: [],
  recurrence_end: "",
}

function formatDays(days: number[] | null): string {
  return (days ?? []).map((d) => DAY_NAMES[d] ?? "?").join(", ")
}

function formatSelect(options: Option[]) {
  return (
    <SelectContent>
      {options.length === 0 && <div className="px-3 py-2 text-sm text-muted-foreground">Kosong</div>}
      {options.map((o) => (
        <SelectItem key={o.id} value={String(o.id)}>
          {o.name}
        </SelectItem>
      ))}
    </SelectContent>
  )
}

export function SchedulesPage() {
  const [rows, setRows] = useState<ScheduleRow[]>([])
  const [classes, setClasses] = useState<Option[]>([])
  const [courses, setCourses] = useState<(Option & { course_name?: string })[]>([])
  const [locations, setLocations] = useState<Option[]>([])
  const [loading, setLoading] = useState(true)
  const [filterClass, setFilterClass] = useState("all")
  const [filterCourse, setFilterCourse] = useState("all")

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<ScheduleRow | null>(null)
  const [form, setForm] = useState<ScheduleFormState>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState("")
  const [deleting, setDeleting] = useState<ScheduleRow | null>(null)

  const [codeTarget, setCodeTarget] = useState<ScheduleRow | null>(null)
  const [minutesValid, setMinutesValid] = useState("30")
  const [codeResult, setCodeResult] = useState<{ code: string; expires_at: string } | null>(null)
  const [generating, setGenerating] = useState(false)

  const [qrTarget, setQrTarget] = useState<ScheduleRow | null>(null)
  const [qrUrl, setQrUrl] = useState("")
  const [qrLoading, setQrLoading] = useState(false)

  const loadOptions = useCallback(async () => {
    try {
      const [classEnv, courseEnv, locEnv] = await Promise.all([
        api<Option[]>("/classes"),
        api<{ id: number; course_name: string; course_code: string }[]>("/courses"),
        api<LocationRow[]>("/locations"),
      ])
      setClasses(classEnv.data ?? [])
      setCourses(
        (courseEnv.data ?? []).map((c) => ({
          id: c.id,
          name: `${c.course_code} - ${c.course_name}`,
        })),
      )
      setLocations(locEnv.data ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat opsi")
    }
  }, [])

  const loadSchedules = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (filterClass !== "all") params.set("class_id", filterClass)
      if (filterCourse !== "all") params.set("course_id", filterCourse)
      const env = await api<ScheduleRow[]>(`/schedules?${params}`)
      setRows(env.data ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat jadwal")
    } finally {
      setLoading(false)
    }
  }, [filterClass, filterCourse])

  useEffect(() => {
    loadOptions()
  }, [loadOptions])
  useEffect(() => {
    loadSchedules()
  }, [loadSchedules])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setFormError("")
    setDialogOpen(true)
  }

  const openEdit = (row: ScheduleRow) => {
    setEditing(row)
    setForm({
      class_id: String(row.class_id),
      course_id: String(row.course_id),
      location_id: String(row.location_id),
      start_time: toDatetimeLocal(row.start_time),
      end_time: toDatetimeLocal(row.end_time),
      is_recurring: row.is_recurring,
      repeat_days: row.repeat_days ?? [],
      recurrence_end: row.recurrence_end ?? "",
    })
    setFormError("")
    setDialogOpen(true)
  }

  const toggleDay = (day: number) => {
    setForm((f) => ({
      ...f,
      repeat_days: f.repeat_days.includes(day)
        ? f.repeat_days.filter((d) => d !== day)
        : [...f.repeat_days, day].sort((a, b) => a - b),
    }))
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!form.class_id || !form.course_id || !form.location_id) {
      setFormError("Kelas, mata kuliah, dan lokasi wajib dipilih.")
      return
    }
    if (form.is_recurring && form.repeat_days.length === 0) {
      setFormError("Pilih minimal satu hari pengulangan.")
      return
    }
    setSaving(true)
    setFormError("")
    const body = {
      class_id: Number(form.class_id),
      course_id: Number(form.course_id),
      location_id: Number(form.location_id),
      start_time: form.start_time.replace("T", " ") + ":00",
      end_time: form.end_time.replace("T", " ") + ":00",
      is_recurring: form.is_recurring,
      repeat_days: form.is_recurring ? form.repeat_days : [],
      recurrence_end:
        form.is_recurring && form.recurrence_end ? form.recurrence_end : null,
    }
    try {
      if (editing) {
        await api(`/schedules/${editing.id}`, { method: "PUT", body: JSON.stringify(body) })
        toast.success("Jadwal berhasil diperbarui")
      } else {
        await api("/schedules", { method: "POST", body: JSON.stringify(body) })
        toast.success("Jadwal berhasil ditambahkan")
      }
      setDialogOpen(false)
      loadSchedules()
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Gagal menyimpan jadwal")
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleting) return
    try {
      await api(`/schedules/${deleting.id}`, { method: "DELETE" })
      toast.success("Jadwal berhasil dihapus")
      loadSchedules()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus")
    } finally {
      setDeleting(null)
    }
  }

  const openGenerateCode = (row: ScheduleRow) => {
    setCodeTarget(row)
    setCodeResult(null)
    setMinutesValid("30")
  }

  const generateCode = async () => {
    if (!codeTarget) return
    setGenerating(true)
    try {
      const body = minutesValid ? { minutes_valid: Number(minutesValid) } : {}
      const env = await api<{ code: string; expires_at: string; minutes_valid: number }>(
        `/schedules/${codeTarget.id}/generate-code`,
        { method: "POST", body: JSON.stringify(body) },
      )
      if (env.data) setCodeResult(env.data)
      loadSchedules()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal membuat kode")
    } finally {
      setGenerating(false)
    }
  }

  const openQr = async (row: ScheduleRow) => {
    setQrTarget(row)
    setQrUrl("")
    if (!row.has_qr) {
      await rotateQr(row.id)
      return
    }
    await loadQr(row.id)
  }

  const loadQr = async (id: number) => {
    setQrLoading(true)
    try {
      const blob = await apiBlob(`/schedules/${id}/qr.png`)
      setQrUrl(URL.createObjectURL(blob))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat QR")
    } finally {
      setQrLoading(false)
    }
  }

  const rotateQr = async (id: number) => {
    setQrLoading(true)
    try {
      await api(`/schedules/${id}/generate-qr`, { method: "POST", body: "{}" })
      toast.success("QR token baru dibuat")
      await loadQr(id)
      loadSchedules()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal rotate QR")
      setQrLoading(false)
    }
  }

  return (
    <AppLayout title="Jadwal">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={filterClass} onValueChange={setFilterClass}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="Semua kelas" />
          </SelectTrigger>
          {formatSelect(classes)}
        </Select>
        <Select value={filterCourse} onValueChange={setFilterCourse}>
          <SelectTrigger className="w-56">
            <SelectValue placeholder="Semua mata kuliah" />
          </SelectTrigger>
          {formatSelect(courses)}
        </Select>
        <Button variant="outline" onClick={loadSchedules}>
          Refresh
        </Button>
        <Button className="ml-auto" onClick={openCreate}>
          <Plus />
          Tambah Jadwal
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Mata Kuliah</TableHead>
              <TableHead>Kelas</TableHead>
              <TableHead>Lokasi</TableHead>
              <TableHead>Waktu</TableHead>
              <TableHead>Kode</TableHead>
              <TableHead className="w-52 text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <p className="font-medium">{row.course}</p>
                    <p className="font-mono text-xs text-muted-foreground">{row.course_code}</p>
                  </TableCell>
                  <TableCell>{row.class}</TableCell>
                  <TableCell>{row.location}</TableCell>
                  <TableCell>
                    <p className="whitespace-nowrap text-sm">{formatDateTime(row.start_time)}</p>
                    <p className="text-xs text-muted-foreground">s/d {formatDateTime(row.end_time)}</p>
                    {row.is_recurring && (
                      <Badge variant="outline" className="mt-1">
                        Berulang {formatDays(row.repeat_days)}
                        {row.recurrence_end ? ` · s/d ${row.recurrence_end}` : ""}
                      </Badge>
                    )}
                    {row.is_active && (
                      <Badge className="mt-1 bg-emerald-600 hover:bg-emerald-600">Berlangsung</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {row.attendance_code ? (
                      <div>
                        <p className="font-mono font-bold tracking-widest">
                          {row.attendance_code}
                        </p>
                        <p className="text-[11px] text-muted-foreground">
                          s/d {formatDateTime(row.code_expires_at)}
                        </p>
                      </div>
                    ) : row.has_qr ? (
                      <Badge variant="outline">
                        <QrCode /> QR aktif
                      </Badge>
                    ) : (
                      "-"
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => openEdit(row)}>
                      Edit
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => openGenerateCode(row)}>
                      <KeyRound />
                      Kode
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => openQr(row)}>
                      <QrCode />
                      QR
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setDeleting(row)}
                    >
                      Hapus
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            {!loading && rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground">
                  Belum ada jadwal.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Dialog tambah jadwal */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Jadwal" : "Tambah Jadwal"}</DialogTitle>
            <DialogDescription>Waktu dalam zona Asia/Jakarta.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Kelas</Label>
                <Select
                  value={form.class_id}
                  onValueChange={(v) => setForm({ ...form, class_id: v })}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Pilih kelas" />
                  </SelectTrigger>
                  {formatSelect(classes)}
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Mata Kuliah</Label>
                <Select
                  value={form.course_id}
                  onValueChange={(v) => setForm({ ...form, course_id: v })}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Pilih mata kuliah" />
                  </SelectTrigger>
                  {formatSelect(courses)}
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Lokasi</Label>
              <Select
                value={form.location_id}
                onValueChange={(v) => setForm({ ...form, location_id: v })}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pilih lokasi" />
                </SelectTrigger>
                {formatSelect(locations)}
              </Select>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="s-start">Mulai</Label>
                <Input
                  id="s-start"
                  type="datetime-local"
                  value={form.start_time}
                  onChange={(e) => setForm({ ...form, start_time: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="s-end">Selesai</Label>
                <Input
                  id="s-end"
                  type="datetime-local"
                  value={form.end_time}
                  onChange={(e) => setForm({ ...form, end_time: e.target.value })}
                  required
                />
              </div>
            </div>
            <div className="space-y-3 rounded-md border p-3">
              <label className="flex items-center gap-2 text-sm font-medium">
                <Checkbox
                  checked={form.is_recurring}
                  onCheckedChange={(v) => setForm({ ...form, is_recurring: v === true })}
                />
                Jadwal berulang mingguan
              </label>
              {form.is_recurring && (
                <>
                  <div className="flex flex-wrap gap-1.5">
                    {DAY_NAMES.map((name, day) => (
                      <button
                        key={day}
                        type="button"
                        onClick={() => toggleDay(day)}
                        className={`rounded-md border px-2.5 py-1 text-xs transition-colors ${
                          form.repeat_days.includes(day)
                            ? "border-primary bg-primary text-primary-foreground"
                            : "hover:bg-muted"
                        }`}
                      >
                        {name}
                      </button>
                    ))}
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="s-rec-end" className="text-muted-foreground">
                      Ulangi sampai (opsional)
                    </Label>
                    <Input
                      id="s-rec-end"
                      type="date"
                      value={form.recurrence_end}
                      onChange={(e) => setForm({ ...form, recurrence_end: e.target.value })}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Waktu mulai/selesai di atas menjadi jam tetap mingguan; tanggalnya adalah
                    kemunculan pertama.
                  </p>
                </>
              )}
            </div>
            {formError && (
              <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {formError}
              </p>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                Batal
              </Button>
              <Button type="submit" disabled={saving}>
                {saving && <Spinner className="size-4" />}
                Simpan
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Dialog generate kode */}
      <Dialog
        open={codeTarget != null}
        onOpenChange={(open) => !open && setCodeTarget(null)}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Kode Absensi — {codeTarget?.course}</DialogTitle>
            <DialogDescription>
              Berlaku {minutesValid || 30} menit sejak dibuat.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {codeResult && (
              <div className="rounded-lg border bg-muted/40 p-4 text-center">
                <p className="font-mono text-3xl font-bold tracking-[0.3em] text-primary">
                  {codeResult.code}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  berlaku sampai {formatDateTime(codeResult.expires_at)}
                </p>
              </div>
            )}
            <div className="flex items-center gap-3">
              <Label htmlFor="g-minutes">Menit berlaku</Label>
              <Input
                id="g-minutes"
                type="number"
                min="1"
                max="1440"
                value={minutesValid}
                onChange={(e) => setMinutesValid(e.target.value)}
                className="w-28"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCodeTarget(null)}>
              Tutup
            </Button>
            <Button onClick={generateCode} disabled={generating}>
              {generating ? <Spinner className="size-4" /> : <RefreshCw />}
              Generate / Rotate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog QR */}
      <Dialog open={qrTarget != null} onOpenChange={(open) => !open && setQrTarget(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>QR Absensi — {qrTarget?.course}</DialogTitle>
            <DialogDescription>Tunjukkan QR ini ke mahasiswa.</DialogDescription>
          </DialogHeader>
          <div className="flex min-h-64 items-center justify-center rounded-lg border bg-white p-4">
            {qrLoading && <Spinner className="size-6" />}
            {!qrLoading && qrUrl && <img src={qrUrl} alt="QR absensi" className="max-w-full" />}
          </div>
          <DialogFooter>
            {qrTarget && (
              <Button variant="outline" onClick={() => rotateQr(qrTarget.id)} disabled={qrLoading}>
                <RefreshCw />
                Rotate Token
              </Button>
            )}
            <Button variant="secondary" onClick={() => setQrTarget(null)}>
              Tutup
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleting != null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus jadwal?</AlertDialogTitle>
            <AlertDialogDescription>
              Jadwal {deleting?.course} ({deleting?.class}) akan dihapus.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AppLayout>
  )
}
