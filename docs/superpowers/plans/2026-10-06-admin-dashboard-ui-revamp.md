# Admin Dashboard UI Revamp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the existing React admin panel UI (AppLayout, AppSidebar, AppHeader, DashboardPage) by integrating partial components and styling from the `shadcn-dashboard-landing-template`, without modifying business logic.

**Architecture:** We will copy essential components from the scratch template folder, integrate them into our `AppLayout` wrapper, tweak CSS variables, and apply new CSS grid layouts to our dashboard metrics.

**Tech Stack:** React 19, Vite, shadcn/ui, Tailwind CSS

**Spec:** `docs/superpowers/specs/2026-10-06-admin-dashboard-ui-revamp-design.md`

## Global Constraints

- Partial Adoption only: do not copy ThemeCustomizer, Pro upgrade buttons, or landing pages.
- Business Logic Preservation: Do not touch `src/lib/api.ts`, `src/lib/auth.tsx`, or any Go backend code.

## Review Focus

- Sidebar navigation links might break if route paths are mistyped during mapping.
- Dashboard data fetching might fail if component wrappers accidentally drop React hooks or state.
- Missing shadcn UI dependencies (like Breadcrumbs or Avatar) might cause build errors if not installed.

---

### Task 1: Setup Additional shadcn/ui Dependencies

**Files:**
- Create: `web/admin/src/components/ui/breadcrumb.tsx` (via shadcn CLI)
- Create: `web/admin/src/components/ui/avatar.tsx` (via shadcn CLI)

**Interfaces:**
- Consumes: Existing shadcn setup
- Produces: Base UI components for the layout

- [ ] **Step 1: Install Breadcrumb**
Run: `cd web/admin && npx shadcn@latest add breadcrumb`
Expected: Component created successfully

- [ ] **Step 2: Install Avatar**
Run: `cd web/admin && npx shadcn@latest add avatar`
Expected: Component created successfully

- [ ] **Step 3: Commit**
```bash
git add web/admin/src/components/ui
git commit -m "chore: add shadcn breadcrumb and avatar components"
```

### Task 2: Migrate CSS Variables and Utilities

**Files:**
- Modify: `web/admin/src/index.css`

**Interfaces:**
- Produces: CSS Variables (`--sidebar-width`, `--sidebar-width-icon`, `--header-height`)

- [ ] **Step 1: Add layout variables to root CSS**
In `web/admin/src/index.css`, append these to the `:root` block:
```css
  --sidebar-width: 16rem;
  --sidebar-width-icon: 3rem;
  --header-height: calc(var(--spacing) * 14);
```

- [ ] **Step 2: Commit**
```bash
git add web/admin/src/index.css
git commit -m "style: add template css variables"
```

### Task 3: Migrate Layout Components (Header & Sidebar)

**Files:**
- Modify: `web/admin/src/components/app-header.tsx`
- Modify: `web/admin/src/components/app-sidebar.tsx`
- Modify: `web/admin/src/components/app-layout.tsx`

**Interfaces:**
- Consumes: CSS variables from Task 2, base shadcn components from Task 1.

- [ ] **Step 1: Update AppHeader**
Refactor `AppHeader` to include breadcrumbs and a polished structure similar to the template's `SiteHeader`, minus the theme togglers. Ensure `SidebarTrigger` is still present for mobile layout.

- [ ] **Step 2: Update AppSidebar**
Refactor `AppSidebar` to use the template's sidebar styling structure (using `SidebarGroup`, `SidebarMenu`, `SidebarMenuButton`). Hardcode the navigation data to our existing routes (`/`, `/users`, `/classes`, `/courses`, `/locations`, `/schedules`, `/attendances`).

- [ ] **Step 3: Update AppLayout wrapper**
Adjust `AppLayout` to match `BaseLayout`'s wrapping div structure:
```tsx
      <SidebarInset>
        <AppHeader title={title} />
        <div className="flex flex-1 flex-col">
           <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 lg:px-6">
              {children}
           </div>
        </div>
      </SidebarInset>
```

- [ ] **Step 4: Verify build**
Run: `cd web/admin && npm run build`
Expected: Build succeeds without type or unresolved import errors.

- [ ] **Step 5: Commit**
```bash
git add web/admin/src/components
git commit -m "feat: integrate template app layout and sidebar"
```

### Task 4: Polish Dashboard Page

**Files:**
- Modify: `web/admin/src/pages/dashboard.tsx`

**Interfaces:**
- Consumes: AppLayout from Task 3.

- [ ] **Step 1: Update Dashboard Grid**
Modify `DashboardPage` to use the template's grid styling for metric cards (e.g., `grid gap-4 md:grid-cols-2 lg:grid-cols-4`). Ensure the existing data fetching hooks (`useDashboardStats`) are perfectly preserved.

- [ ] **Step 2: Run verification**
Run: `cd web/admin && npm run build`
Expected: Build succeeds.

- [ ] **Step 3: Commit**
```bash
git add web/admin/src/pages/dashboard.tsx
git commit -m "feat: apply template grid styling to dashboard metrics"
```
