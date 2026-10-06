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
import { Checkbox } from "@/components/ui/checkbox"
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

interface ClassRow {
  id: number
  name: string
  academic_year: string
  students?: { id: number; name: string }[]
}

interface StudentRow {
  id: number
  name: string
  identity_number: string
}

interface ClassFormState {
  name: string
  academic_year: string
  user_ids: number[]
}

const emptyForm: ClassFormState = { name: "", academic_year: "", user_ids: [] }

export function ClassesPage() {
  const [rows, setRows] = useState<ClassRow[]>([])
  const [students, setStudents] = useState<StudentRow[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<ClassRow | null>(null)
  const [form, setForm] = useState<ClassFormState>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState("")
  const [deleting, setDeleting] = useState<ClassRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [classEnv, studentEnv] = await Promise.all([
        api<ClassRow[]>("/classes"),
        api<StudentRow[]>("/users?role=mahasiswa"),
      ])
      setRows(classEnv.data ?? [])
      setStudents(studentEnv.data ?? [])
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

  const openEdit = (row: ClassRow) => {
    setEditing(row)
    setForm({
      name: row.name,
      academic_year: row.academic_year,
      user_ids: (row.students ?? []).map((s) => s.id),
    })
    setFormError("")
    setDialogOpen(true)
  }

  const toggleStudent = (id: number) => {
    setForm((prev) => ({
      ...prev,
      user_ids: prev.user_ids.includes(id)
        ? prev.user_ids.filter((uid) => uid !== id)
        : [...prev.user_ids, id],
    }))
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError("")
    try {
      if (editing) {
        await api(`/classes/${editing.id}`, { method: "PUT", body: JSON.stringify(form) })
        toast.success("Kelas berhasil diperbarui")
      } else {
        await api("/classes", { method: "POST", body: JSON.stringify(form) })
        toast.success("Kelas berhasil ditambahkan")
      }
      setDialogOpen(false)
      load()
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Gagal menyimpan kelas")
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleting) return
    try {
      await api(`/classes/${deleting.id}`, { method: "DELETE" })
      toast.success("Kelas berhasil dihapus")
      load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus")
    } finally {
      setDeleting(null)
    }
  }

  return (
    <AppLayout title="Kelas">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Kelola kelas dan anggotanya.</p>
        <Button onClick={openCreate}>
          <Plus />
          Tambah Kelas
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama Kelas</TableHead>
              <TableHead>Tahun Ajaran</TableHead>
              <TableHead>Jumlah Mahasiswa</TableHead>
              <TableHead className="w-28 text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && (
              <TableRow>
                <TableCell colSpan={4} className="py-8 text-center">
                  <Spinner className="mx-auto size-5" />
                </TableCell>
              </TableRow>
            )}
            {!loading &&
              rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-medium">{row.name}</TableCell>
                  <TableCell>{row.academic_year}</TableCell>
                  <TableCell>{(row.students ?? []).length}</TableCell>
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
                <TableCell colSpan={4} className="text-center text-muted-foreground">
                  Belum ada kelas.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Kelas" : "Tambah Kelas"}</DialogTitle>
            <DialogDescription>Pilih mahasiswa untuk diikutkan ke kelas ini.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="c-name">Nama Kelas</Label>
                <Input
                  id="c-name"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="c-year">Tahun Ajaran</Label>
                <Input
                  id="c-year"
                  placeholder="2025/2026"
                  value={form.academic_year}
                  onChange={(e) => setForm({ ...form, academic_year: e.target.value })}
                  required
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Mahasiswa</Label>
              <div className="max-h-48 space-y-2 overflow-y-auto rounded-md border p-3">
                {students.length === 0 && (
                  <p className="text-sm text-muted-foreground">Belum ada mahasiswa.</p>
                )}
                {students.map((s) => (
                  <label key={s.id} className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={form.user_ids.includes(s.id)}
                      onCheckedChange={() => toggleStudent(s.id)}
                    />
                    {s.name}
                    <span className="font-mono text-xs text-muted-foreground">
                      {s.identity_number}
                    </span>
                  </label>
                ))}
              </div>
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
            <AlertDialogTitle>Hapus kelas?</AlertDialogTitle>
            <AlertDialogDescription>
              Kelas {deleting?.name} akan dihapus.
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
