import client from './client';
import type { Project, PaginatedResponse } from '@/types';

export const projectApi = {
  list(params?: Record<string, unknown>): Promise<PaginatedResponse<Project>> {
    return client.get('/projects', { params });
  },

  get(id: number): Promise<Project> {
    return client.get(`/projects/${id}`);
  },

  create(data: { name: string; description?: string; doc_links?: unknown[]; member_ids?: number[] }): Promise<Project> {
    return client.post('/projects', data);
  },

  update(id: number, data: Record<string, unknown>): Promise<Project> {
    return client.put(`/projects/${id}`, data);
  },

  archive(id: number): Promise<{ id: number; status: string }> {
    return client.put(`/projects/${id}/archive`);
  },

  addMembers(id: number, userIds: number[], role: string): Promise<{ added: unknown[]; skipped: unknown[] }> {
    return client.post(`/projects/${id}/members`, { user_ids: userIds, role });
  },

  removeMember(id: number, userId: number): Promise<{ message: string }> {
    return client.delete(`/projects/${id}/members/${userId}`);
  },
};
