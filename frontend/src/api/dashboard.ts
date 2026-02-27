import client from './client';
import type { DashboardStats, MyTasks } from '@/types';

export const dashboardApi = {
  getStats(): Promise<DashboardStats> {
    return client.get('/dashboard/stats');
  },

  getMyTasks(): Promise<MyTasks> {
    return client.get('/dashboard/my-tasks');
  },
};
