import client from './client';
import type { Repository, PaginatedResponse, ConnectionTestResult } from '@/types';

export const repositoryApi = {
  listByProject(projectId: number, params?: Record<string, unknown>): Promise<PaginatedResponse<Repository>> {
    return client.get(`/projects/${projectId}/repos`, { params });
  },

  get(id: number): Promise<Repository> {
    return client.get(`/repos/${id}`);
  },

  create(projectId: number, data: Record<string, unknown>): Promise<Repository> {
    return client.post(`/projects/${projectId}/repos`, data);
  },

  update(id: number, data: Record<string, unknown>): Promise<Repository> {
    return client.put(`/repos/${id}`, data);
  },

  delete(id: number): Promise<{ message: string }> {
    return client.delete(`/repos/${id}`);
  },

  testConnection(id: number): Promise<ConnectionTestResult> {
    return client.post(`/repos/${id}/test-connection`);
  },

  triggerAnalysis(id: number): Promise<{ id: number; analysis_status: string; message: string }> {
    return client.post(`/repos/${id}/analyze`);
  },

  getAnalysis(id: number): Promise<{ analysis_status: string; analyzed_at: string | null; result: unknown; error?: string }> {
    return client.get(`/repos/${id}/analysis`);
  },
};
