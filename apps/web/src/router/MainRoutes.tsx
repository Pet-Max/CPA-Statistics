import { Navigate, useRoutes, type Location } from 'react-router-dom';
import { DashboardPage } from '@/pages/DashboardPage';
import { UsagePage } from '@/pages/UsagePage';
import { RequestDetailsPage } from '@/pages/RequestDetailsPage';
import { DataPage } from '@/pages/DataPage';

const mainRoutes = [
  { path: '/', element: <DashboardPage /> },
  { path: '/overview', element: <DashboardPage /> },
  { path: '/dashboard', element: <Navigate to="/overview" replace /> },
  { path: '/usage', element: <UsagePage /> },
  { path: '/monitoring', element: <Navigate to="/usage" replace /> },
  { path: '/request-details', element: <RequestDetailsPage /> },
  { path: '/data', element: <DataPage /> },
  { path: '*', element: <Navigate to="/overview" replace /> },
];

export function MainRoutes({ location }: { location?: Location }) {
  return useRoutes(mainRoutes, location);
}
