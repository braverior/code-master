import client from './client';

export const feishuApi = {
  resolveDoc(url: string) {
    return client.post('/feishu/doc/resolve', { url }) as Promise<{
      title: string;
      document_id: string;
      url: string;
    }>;
  },
};
