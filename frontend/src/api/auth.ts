import client from './client';
import type { User } from '@/types';

export const authApi = {
  getMe(): Promise<User> {
    return client.get('/auth/me');
  },

  selectRole(role: string, userId?: number): Promise<User> {
    return client.put('/auth/role', { role, user_id: userId });
  },

  refreshToken(): Promise<{ token: string; expire_at: string }> {
    return client.post('/auth/refresh');
  },
};
