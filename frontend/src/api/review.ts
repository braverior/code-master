import client from './client';
import type { Review, PaginatedResponse } from '@/types';

export const reviewApi = {
  get(id: number): Promise<Review> {
    return client.get(`/reviews/${id}`);
  },

  getPending(params?: Record<string, unknown>): Promise<PaginatedResponse<Review>> {
    return client.get('/reviews/pending', { params });
  },

  list(params?: Record<string, unknown>): Promise<PaginatedResponse<Review>> {
    return client.get('/reviews/list', { params });
  },

  submitHumanReview(id: number, data: { comment?: string; status: string }): Promise<Review> {
    return client.put(`/reviews/${id}/human`, data);
  },

  createMergeRequest(id: number): Promise<{
    review_id: number;
    merge_request_id: string;
    merge_request_url: string;
    merge_status: string;
  }> {
    return client.post(`/reviews/${id}/merge-request`);
  },

  getMergeRequest(id: number): Promise<{
    merge_request_id?: string;
    merge_request_url: string | null;
    merge_status: string;
    title?: string;
    ci_status?: string;
    merged_at?: string;
  }> {
    return client.get(`/reviews/${id}/merge-request`);
  },
};
