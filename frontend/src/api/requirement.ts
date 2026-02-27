import client from './client';
import type { Requirement, PaginatedResponse } from '@/types';

export const requirementApi = {
  list(params?: Record<string, unknown>): Promise<PaginatedResponse<Requirement>> {
    return client.get('/requirements', { params });
  },

  listByProject(projectId: number, params?: Record<string, unknown>): Promise<PaginatedResponse<Requirement>> {
    return client.get(`/projects/${projectId}/requirements`, { params });
  },

  get(id: number): Promise<Requirement> {
    return client.get(`/requirements/${id}`);
  },

  create(projectId: number, data: Record<string, unknown>): Promise<Requirement> {
    return client.post(`/projects/${projectId}/requirements`, data);
  },

  update(id: number, data: Record<string, unknown>): Promise<Requirement> {
    return client.put(`/requirements/${id}`, data);
  },

  delete(id: number, force?: boolean): Promise<{ message: string }> {
    return client.delete(`/requirements/${id}`, { data: { force } });
  },

  generate(id: number, data?: { extra_context?: string; source_branch?: string }): Promise<{
    task_id: number;
    status: string;
    source_branch: string;
    target_branch: string;
    queue_position: number;
  }> {
    return client.post(`/requirements/${id}/generate`, data || {});
  },

  manualSubmit(id: number, data?: { source_branch?: string; commit_message?: string; commit_url?: string }): Promise<{
    task_id: number;
    status: string;
    source_branch: string;
    target_branch: string;
  }> {
    return client.post(`/requirements/${id}/manual-submit`, data || {});
  },

  getCodegenTasks(id: number, params?: Record<string, unknown>): Promise<PaginatedResponse<unknown>> {
    return client.get(`/requirements/${id}/codegen-tasks`, { params });
  },
};
