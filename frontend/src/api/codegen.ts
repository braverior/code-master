import client from './client';
import type { CodeGenTask, DiffFile } from '@/types';

export const codegenApi = {
  get(id: number): Promise<CodeGenTask> {
    return client.get(`/codegen/${id}`);
  },

  cancel(id: number): Promise<{ id: number; status: string; cancelled_at: string }> {
    return client.post(`/codegen/${id}/cancel`);
  },

  getDiff(id: number, params?: { file?: string; format?: string }): Promise<{
    target_branch: string;
    base_branch: string;
    files: DiffFile[];
  }> {
    return client.get(`/codegen/${id}/diff`, { params });
  },

  getLog(id: number, params?: { offset?: number; limit?: number }): Promise<{
    task_id: number;
    status: string;
    total_events: number;
    events: unknown[];
    has_more: boolean;
  }> {
    return client.get(`/codegen/${id}/log`, { params });
  },

  triggerReview(id: number, data: { reviewer_ids: number[] }): Promise<{ review_id: number; ai_status: string; message: string }> {
    return client.post(`/codegen/${id}/review`, data);
  },

  getReview(id: number): Promise<unknown> {
    return client.get(`/codegen/${id}/review`);
  },
};
