import { useCallback, useEffect, useState } from "react"
import type { FormEvent } from "react"
import { api } from "@/lib/api"
import { ApiError } from "@/lib/api"
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
import { Spinner } from "@/components/ui/spinner"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Plus } from "lucide-react"
import { toast } from "sonner"

interface CourseRow {
  id: number
  course_name: string
  course_code: string
  lecturer_name: string
  location_room: string | null
}

interface CourseFormState {
  course_name: string
  course_code: string
  lecturer_name: string
  location_room: string
}

const emptyForm: CourseFormState = {
  course_name: "",
  course_code: "",
  lecturer_name: "",
  location_room: "",
}

export function CoursesPage() {
  const [rows, setRows] = useState<CourseRow[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<CourseRow | null>(null)
  const [form, setForm] = useState<CourseFormState>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState("")
  const [deleting, setDeleting] = useState<CourseRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const env = await api<CourseRow[]>("/courses")
      setRows(env.data ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat data")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setFormError("")
    setDialogOpen(true)
  }

  const openEdit = (row: CourseRow) => {
    setEditing(row)
    setForm({
      course_name: row.course_name,
      course_code: row.course_code,
      lecturer_name: row.lecturer_name,
      location_room: row.location_room ?? "",
    })
    setFormError("")
    setDialogOpen(true)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError("")
    try {
      if (editing) {
        await api(`/courses/${editing.id}`, { method: "PUT", body: JSON.stringify(form) })
        toast.success("Mata kuliah berhasil diperbarui")
      } else {
        await api("/courses", { method: "POST", body: JSON.stringify(form) })
        toast.success("Mata kuliah berhasil ditambahkan")
      }
      setDialogOpen(false)
      load()
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Gagal menyimpan mata kuliah")
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleting) return
    try {
      await api(`/courses/${deleting.id}`, { method: "DELETE" })
      toast.success("Mata kuliah berhasil dihapus")
      load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus")
    } finally {
      setDeleting(null)
    }
  }

  return (
    <AppLayout title="Mata Kuliah">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Kode mata kuliah otomatis di-uppercase.</p>
        <Button onClick={openCreate}>
          <Plus />
          Tambah Mata Kuliah
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Kode</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Dosen Pengampu</TableHead>
              <TableHead>Ruang</TableHead>
              <TableHead className="w-28 text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && (
              <TableRow>
                <TableCell colSpan={5} className="py-8 text-center">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <span className="rounded bg-muted px-2 py-0.5 font-mono text-xs font-medium">
                      {row.course_code}
                    </span>
                  </TableCell>
                  <TableCell className="font-medium">{row.course_name}</TableCell>
                  <TableCell>{row.lecturer_name}</TableCell>
                  <TableCell>{row.location_room ?? "-"}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => openEdit(row)}>
                      Edit
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
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  Belum ada mata kuliah.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Mata Kuliah" : "Tambah Mata Kuliah"}</DialogTitle>
            <DialogDescription>Kode harus unik antar mata kuliah.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="co-code">Kode</Label>
                <Input
                  id="co-code"
                  value={form.course_code}
                  onChange={(e) =>
                    setForm({ ...form, course_code: e.target.value.toUpperCase() })
                  }
                  placeholder="IF301"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="co-room">Ruang (opsional)</Label>
                <Input
                  id="co-room"
                  value={form.location_room}
                  onChange={(e) => setForm({ ...form, location_room: e.target.value })}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="co-name">Nama Mata Kuliah</Label>
              <Input
                id="co-name"
                value={form.course_name}
                onChange={(e) => setForm({ ...form, course_name: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="co-lecturer">Dosen Pengampu</Label>
              <Input
                id="co-lecturer"
                value={form.lecturer_name}
                onChange={(e) => setForm({ ...form, lecturer_name: e.target.value })}
                required
              />
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

      <AlertDialog open={deleting != null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus mata kuliah?</AlertDialogTitle>
            <AlertDialogDescription>
              {deleting?.course_name} ({deleting?.course_code}) akan dihapus.
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
