import client from './client';
import type { User, PaginatedResponse, OperationLog } from '@/types';

export const adminApi = {
  listUsers(params?: Record<string, unknown>): Promise<PaginatedResponse<User>> {
    return client.get('/admin/users', { params });
  },

  updateUserRole(id: number, role: string): Promise<User> {
    return client.put(`/admin/users/${id}/role`, { role });
  },

  toggleUserAdmin(id: number, isAdmin: boolean): Promise<User> {
    return client.put(`/admin/users/${id}/admin`, { is_admin: isAdmin });
  },

  updateUserStatus(id: number, status: number): Promise<User> {
    return client.put(`/admin/users/${id}/status`, { status });
  },

  getOperationLogs(params?: Record<string, unknown>): Promise<PaginatedResponse<OperationLog>> {
    return client.get('/admin/operation-logs', { params });
  },

  searchUsers(params: { keyword: string; role?: string; exclude_project_id?: number; limit?: number }): Promise<User[]> {
    return client.get('/users/search', { params });
  },
};
