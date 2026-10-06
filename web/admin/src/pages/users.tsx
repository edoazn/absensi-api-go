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
import { Plus } from "lucide-react"
import { toast } from "sonner"

interface UserRow {
  id: number
  name: string
  identity_number: string
  email: string | null
  role: string
  classes?: { id: number; name: string }[]
}

interface ClassRow {
  id: number
  name: string
}

interface UserFormState {
  name: string
  identity_number: string
  email: string
  password: string
  role: string
  class_ids: number[]
}

const emptyForm: UserFormState = {
  name: "",
  identity_number: "",
  email: "",
  password: "",
  role: "mahasiswa",
  class_ids: [],
}

export function UsersPage() {
  const [rows, setRows] = useState<UserRow[]>([])
  const [classes, setClasses] = useState<ClassRow[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<UserRow | null>(null)
  const [form, setForm] = useState<UserFormState>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState("")
  const [deleting, setDeleting] = useState<UserRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [usersEnv, classesEnv] = await Promise.all([
        api<UserRow[]>("/users"),
        api<ClassRow[]>("/classes"),
      ])
      setRows(usersEnv.data ?? [])
      setClasses(classesEnv.data ?? [])
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

  const openEdit = (row: UserRow) => {
    setEditing(row)
    setForm({
      name: row.name,
      identity_number: row.identity_number,
      email: row.email ?? "",
      password: "",
      role: row.role,
      class_ids: (row.classes ?? []).map((c) => c.id),
    })
    setFormError("")
    setDialogOpen(true)
  }

  const toggleClass = (id: number) => {
    setForm((prev) => ({
      ...prev,
      class_ids: prev.class_ids.includes(id)
        ? prev.class_ids.filter((cid) => cid !== id)
        : [...prev.class_ids, id],
    }))
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError("")
    const body: Record<string, unknown> = {
      name: form.name,
      identity_number: form.identity_number,
      email: form.email || null,
      role: form.role,
      class_ids: form.role === "mahasiswa" ? form.class_ids : [],
    }
    if (form.password) body.password = form.password

    try {
      if (editing) {
        await api(`/users/${editing.id}`, { method: "PUT", body: JSON.stringify(body) })
        toast.success("Pengguna berhasil diperbarui")
      } else {
        body.password = form.password
        await api("/users", { method: "POST", body: JSON.stringify(body) })
        toast.success("Pengguna berhasil ditambahkan")
      }
      setDialogOpen(false)
      load()
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Gagal menyimpan pengguna")
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleting) return
    try {
      await api(`/users/${deleting.id}`, { method: "DELETE" })
      toast.success("Pengguna berhasil dihapus")
      load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus")
    } finally {
      setDeleting(null)
    }
  }

  return (
    <AppLayout title="Pengguna">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Kelola admin dan mahasiswa.</p>
        <Button onClick={openCreate}>
          <Plus />
          Tambah Pengguna
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama</TableHead>
              <TableHead>NIM/NIP</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Kelas</TableHead>
              <TableHead className="w-28 text-right">Aksi</TableHead>
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
                  <TableCell className="font-medium">{row.name}</TableCell>
                  <TableCell className="font-mono text-xs">{row.identity_number}</TableCell>
                  <TableCell>{row.email ?? "-"}</TableCell>
                  <TableCell>
                    {row.role === "admin" ? (
                      <Badge variant="default">Admin</Badge>
                    ) : (
                      <Badge variant="secondary">Mahasiswa</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {(row.classes ?? []).length === 0 && "-"}
                      {(row.classes ?? []).map((c) => (
                        <Badge key={c.id} variant="outline">
                          {c.name}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
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
                <TableCell colSpan={6} className="text-center text-muted-foreground">
                  Belum ada pengguna.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Pengguna" : "Tambah Pengguna"}</DialogTitle>
            <DialogDescription>
              {editing
                ? "Password dikosongkan bila tidak diubah."
                : "Password minimal 6 karakter."}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="u-name">Nama</Label>
                <Input
                  id="u-name"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="u-nim">NIM/NIP</Label>
                <Input
                  id="u-nim"
                  value={form.identity_number}
                  onChange={(e) => setForm({ ...form, identity_number: e.target.value })}
                  required
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="u-email">Email (opsional)</Label>
                <Input
                  id="u-email"
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="u-role">Role</Label>
                <Select
                  value={form.role}
                  onValueChange={(v) => setForm({ ...form, role: v })}
                >
                  <SelectTrigger id="u-role" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="mahasiswa">Mahasiswa</SelectItem>
                    <SelectItem value="admin">Admin</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="u-password">
                Password {editing && "(kosongkan bila tidak diubah)"}
              </Label>
              <Input
                id="u-password"
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required={!editing}
              />
            </div>
            {form.role === "mahasiswa" && (
              <div className="space-y-2">
                <Label>Kelas</Label>
                <div className="max-h-40 space-y-2 overflow-y-auto rounded-md border p-3">
                  {classes.length === 0 && (
                    <p className="text-sm text-muted-foreground">Belum ada kelas.</p>
                  )}
                  {classes.map((c) => (
                    <label key={c.id} className="flex items-center gap-2 text-sm">
                      <Checkbox
                        checked={form.class_ids.includes(c.id)}
                        onCheckedChange={() => toggleClass(c.id)}
                      />
                      {c.name}
                    </label>
                  ))}
                </div>
              </div>
            )}
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
            <AlertDialogTitle>Hapus pengguna?</AlertDialogTitle>
            <AlertDialogDescription>
              {deleting?.name} akan dinonaktifkan (soft delete).
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
