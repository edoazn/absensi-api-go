import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AuthProvider } from "@/lib/auth";
import { RequireAdmin } from "@/components/require-admin";
import { LoginPage } from "@/pages/login";
import { DashboardPage } from "@/pages/dashboard";
import { UsersPage } from "@/pages/users";
import { ClassesPage } from "@/pages/classes";
import { CoursesPage } from "@/pages/courses";
import { LocationsPage } from "@/pages/locations";
import { SchedulesPage } from "@/pages/schedules";
import { AttendancesPage } from "@/pages/attendances";

export default function App() {
  return (
    <BrowserRouter basename={import.meta.env.BASE_URL}>
      <AuthProvider>
        <TooltipProvider delayDuration={200}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<RequireAdmin />}>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/users" element={<UsersPage />} />
              <Route path="/classes" element={<ClassesPage />} />
              <Route path="/courses" element={<CoursesPage />} />
              <Route path="/locations" element={<LocationsPage />} />
              <Route path="/schedules" element={<SchedulesPage />} />
              <Route path="/attendances" element={<AttendancesPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </TooltipProvider>
        <Toaster position="top-center" richColors />
      </AuthProvider>
    </BrowserRouter>
  );
}
