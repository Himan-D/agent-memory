import { Routes, Route } from 'react-router-dom';
import { Providers } from '@/components/providers';
import { DashboardLayout } from '@/layouts/DashboardLayout';
import { AuthLayout } from '@/layouts/AuthLayout';
import { AuthSessionSync } from '@/components/auth/auth-session-sync';

import DashboardPage from '@/pages/dashboard/index';
import AgentsPage from '@/pages/dashboard/agents';
import AlertsPage from '@/pages/dashboard/alerts';
import AnalyticsPage from '@/pages/dashboard/analytics';
import ApiKeysPage from '@/pages/dashboard/api-keys';
import AuditPage from '@/pages/dashboard/audit';
import BillingPage from '@/pages/dashboard/billing';
import ChainsPage from '@/pages/dashboard/chains';
import DocumentsPage from '@/pages/dashboard/documents';
import EntitiesPage from '@/pages/dashboard/entities';
import GroupsPage from '@/pages/dashboard/groups';
import MemoriesPage from '@/pages/dashboard/memories';
import NotificationsPage from '@/pages/dashboard/notifications';
import PlaygroundPage from '@/pages/dashboard/playground';
import ProjectsPage from '@/pages/dashboard/projects';
import SearchPage from '@/pages/dashboard/search';
import SessionsPage from '@/pages/dashboard/sessions';
import SettingsPage from '@/pages/dashboard/settings';
import SkillsPage from '@/pages/dashboard/skills';
import UsersPage from '@/pages/dashboard/users';
import WebhooksPage from '@/pages/dashboard/webhooks';

import SigninPage from '@/pages/auth/signin';
import SignupPage from '@/pages/auth/signup';
import ErrorPage from '@/pages/auth/error';

import DemoPage from '@/pages/demo/index';

import { ThemeProvider } from 'next-themes';

function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
      <Providers>
        <AuthSessionSync />
        <Routes>
          <Route element={<DashboardLayout />}>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/agents" element={<AgentsPage />} />
            <Route path="/alerts" element={<AlertsPage />} />
            <Route path="/analytics" element={<AnalyticsPage />} />
            <Route path="/api-keys" element={<ApiKeysPage />} />
            <Route path="/audit" element={<AuditPage />} />
            <Route path="/billing" element={<BillingPage />} />
            <Route path="/chains" element={<ChainsPage />} />
            <Route path="/documents" element={<DocumentsPage />} />
            <Route path="/entities" element={<EntitiesPage />} />
            <Route path="/groups" element={<GroupsPage />} />
            <Route path="/memories" element={<MemoriesPage />} />
            <Route path="/notifications" element={<NotificationsPage />} />
            <Route path="/playground" element={<PlaygroundPage />} />
            <Route path="/projects" element={<ProjectsPage />} />
            <Route path="/search" element={<SearchPage />} />
            <Route path="/sessions" element={<SessionsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/skills" element={<SkillsPage />} />
            <Route path="/users" element={<UsersPage />} />
            <Route path="/webhooks" element={<WebhooksPage />} />
          </Route>
          <Route path="/auth" element={<AuthLayout />}>
            <Route path="signin" element={<SigninPage />} />
            <Route path="signup" element={<SignupPage />} />
            <Route path="error" element={<ErrorPage />} />
          </Route>
          <Route path="/demo" element={<DemoPage />} />
        </Routes>
      </Providers>
    </ThemeProvider>
  );
}

export default App;
