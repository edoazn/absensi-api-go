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

interface LocationRow {
  id: number
  name: string
  latitude: number
  longitude: number
  radius: number
}

interface LocationFormState {
  name: string
  latitude: string
  longitude: string
  radius: string
}

const emptyForm: LocationFormState = { name: "", latitude: "", longitude: "", radius: "" }

export function LocationsPage() {
  const [rows, setRows] = useState<LocationRow[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<LocationRow | null>(null)
  const [form, setForm] = useState<LocationFormState>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState("")
  const [deleting, setDeleting] = useState<LocationRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const env = await api<LocationRow[]>("/locations")
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

  const openEdit = (row: LocationRow) => {
    setEditing(row)
    setForm({
      name: row.name,
      latitude: String(row.latitude),
      longitude: String(row.longitude),
      radius: String(row.radius),
    })
    setFormError("")
    setDialogOpen(true)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setFormError("")
    const body = {
      name: form.name,
      latitude: Number(form.latitude.replace(",", ".")),
      longitude: Number(form.longitude.replace(",", ".")),
      radius: Number(form.radius),
    }
    try {
      if (editing) {
        await api(`/locations/${editing.id}`, { method: "PUT", body: JSON.stringify(body) })
        toast.success("Lokasi berhasil diperbarui")
      } else {
        await api("/locations", { method: "POST", body: JSON.stringify(body) })
        toast.success("Lokasi berhasil ditambahkan")
      }
      setDialogOpen(false)
      load()
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Gagal menyimpan lokasi")
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleting) return
    try {
      await api(`/locations/${deleting.id}`, { method: "DELETE" })
      toast.success("Lokasi berhasil dihapus")
      load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus")
    } finally {
      setDeleting(null)
    }
  }

  return (
    <AppLayout title="Lokasi">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Radius digunakan untuk validasi absensi geolocation.
        </p>
        <Button onClick={openCreate}>
          <Plus />
          Tambah Lokasi
        </Button>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nama</TableHead>
              <TableHead>Latitude</TableHead>
              <TableHead>Longitude</TableHead>
              <TableHead>Radius</TableHead>
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
                  <TableCell className="font-medium">{row.name}</TableCell>
                  <TableCell className="font-mono text-xs">{row.latitude}</TableCell>
                  <TableCell className="font-mono text-xs">{row.longitude}</TableCell>
                  <TableCell>{row.radius} m</TableCell>
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
                  Belum ada lokasi.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit Lokasi" : "Tambah Lokasi"}</DialogTitle>
            <DialogDescription>
              Latitude -90..90, longitude -180..180, radius dalam meter.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="l-name">Nama Lokasi</Label>
              <Input
                id="l-name"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                required
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="l-lat">Latitude</Label>
                <Input
                  id="l-lat"
                  value={form.latitude}
                  onChange={(e) => setForm({ ...form, latitude: e.target.value })}
                  placeholder="-6.2"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="l-lng">Longitude</Label>
                <Input
                  id="l-lng"
                  value={form.longitude}
                  onChange={(e) => setForm({ ...form, longitude: e.target.value })}
                  placeholder="106.816666"
                  required
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="l-radius">Radius (meter)</Label>
              <Input
                id="l-radius"
                type="number"
                min="1"
                step="any"
                value={form.radius}
                onChange={(e) => setForm({ ...form, radius: e.target.value })}
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
            <AlertDialogTitle>Hapus lokasi?</AlertDialogTitle>
            <AlertDialogDescription>
              Lokasi {deleting?.name} akan dihapus.
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
