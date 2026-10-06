# Admin Dashboard UI Revamp Specification

## 1. Intent & Context
The project currently uses a custom-built React 19 + Vite + shadcn SPA for the Admin Panel (`web/admin`). The human partner found a visually appealing shadcn dashboard template (`shadcnstore/shadcn-dashboard-landing-template`) and wants to adopt its UI/UX into our existing admin panel.

We have decided on a **Partial Adoption** approach to keep the codebase clean. We will cherry-pick the layout and styling improvements while strictly avoiding bloatware (like Theme Customizer, landing pages, or "Upgrade to Pro" buttons). 

## 2. Scope & Constraints
- **In Scope:** 
  - Upgrading the `AppLayout`, `AppSidebar`, and `AppHeader` to match the template's look.
  - Applying the template's CSS variables and grid layout styles to our existing pages (especially the `DashboardPage`).
  - Copying necessary UI components from the template's `vite-version/src/components` that improve the layout.
- **Out of Scope:** 
  - Dynamic Theme Customizer (layout orientation configs, dynamic primary color injection).
  - Landing pages, unused auth mock pages, calendar templates.
  - Changes to the Go backend API, JWT auth logic, or React Router structure.

## 3. Architecture & Components

### 3.1. Layout Changes
- Replace the current simple `SidebarProvider` wrapper in `AppLayout` with the structure inspired by the template's `BaseLayout`, retaining our current `children` props injection.
- Update `AppSidebar` to use the styling and structure from the template, mapping our existing routes (Dashboard, Users, Classes, etc.) into the template's sidebar navigation data structure.
- Update `AppHeader` to mimic the template's `SiteHeader`, including breadcrumbs or user profile dropdowns as styled in the template.

### 3.2. Styling & Dependencies
- Merge the template's `index.css` global variables (like `--sidebar-width`) into `web/admin/src/index.css`.
- Ensure all required shadcn UI components used by the template's layout (e.g., `Breadcrumb`, `DropdownMenu`, `Avatar`) are present and updated in `web/admin/src/components/ui`.

### 3.3. Business Logic Preservation
- `src/lib/api.ts` and `src/lib/auth.tsx` will remain completely untouched.
- `App.tsx` routing will remain untouched.

## 4. Implementation Strategy
1. **Asset Migration:** Copy necessary global CSS, icons, and layout component files from the template repository to `web/admin`.
2. **Layout Integration:** Refactor `AppLayout.tsx`, `AppHeader.tsx`, and `AppSidebar.tsx` using the copied template components.
3. **Dashboard Polish:** Apply the new layout classes and wrapper grids to `DashboardPage.tsx`.
4. **Verification:** Build the React app (`npm run build`) and ensure no TypeScript or routing errors occur, and that the UI renders correctly.
