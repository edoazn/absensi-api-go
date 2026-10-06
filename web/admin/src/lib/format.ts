export function formatDateTime(value: string | null | undefined): string {
  if (!value) return "-"
  return value.replace("T", " ").slice(0, 16)
}

export function formatTimeShort(value: string | null | undefined): string {
  if (!value) return "-"
  return value.slice(11, 16)
}

export function toDatetimeLocal(value: string): string {
  return value.slice(0, 16).replace(" ", "T")
}

export function statusLabel(status: string): string {
  return status === "hadir" ? "Hadir" : "Ditolak"
}

export function methodLabel(method: string): string {
  switch (method) {
    case "geolocation":
      return "Geolocation"
    case "qr_code":
      return "QR Code"
    case "attendance_code":
      return "Kode Manual"
    default:
      return method
  }
}
