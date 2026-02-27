import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from '@/hooks/use-auth';
import { Layout } from '@/components/Layout';
import { PrivateRoute } from '@/components/PrivateRoute';
import { RoleGuard } from '@/components/RoleGuard';
import { LoginPage } from '@/pages/LoginPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { ProjectListPage } from '@/pages/ProjectListPage';
import { ProjectDetailPage } from '@/pages/ProjectDetailPage';
import { RequirementDetailPage } from '@/pages/RequirementDetailPage';
import { RequirementListPage } from '@/pages/RequirementListPage';
import { CodeGenPage } from '@/pages/CodeGenPage';
import { ReviewPage } from '@/pages/ReviewPage';
import { ReviewListPage } from '@/pages/ReviewListPage';
import { AdminUsersPage } from '@/pages/AdminUsersPage';
import { AdminLogsPage } from '@/pages/AdminLogsPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { GuidePage } from '@/pages/GuidePage';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Public routes */}
          <Route path="/login" element={<LoginPage />} />

          {/* Protected routes */}
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Layout />
              </PrivateRoute>
            }
          >
            <Route index element={<DashboardPage />} />
            <Route path="projects" element={<ProjectListPage />} />
            <Route path="projects/:id" element={<ProjectDetailPage />} />
            <Route path="requirements" element={<RequirementListPage />} />
            <Route path="requirements/:id" element={<RequirementDetailPage />} />
            <Route path="codegen/:id" element={<CodeGenPage />} />
            <Route path="reviews" element={<ReviewListPage />} />
            <Route path="reviews/:id" element={<ReviewPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path="guide" element={<GuidePage />} />
            <Route
              path="admin/users"
              element={
                <RoleGuard roles={['admin']}>
                  <AdminUsersPage />
                </RoleGuard>
              }
            />
            <Route
              path="admin/logs"
              element={
                <RoleGuard roles={['admin']}>
                  <AdminLogsPage />
                </RoleGuard>
              }
            />
          </Route>

          {/* Catch-all */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
