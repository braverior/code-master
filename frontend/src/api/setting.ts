import client from './client';

export interface LLMSettings {
  base_url: string;
  api_key: string;
  model: string;
  gitlab_token: string;
}

export const settingApi = {
  getLLM(): Promise<LLMSettings> {
    return client.get('/settings/llm');
  },

  updateLLM(data: LLMSettings): Promise<LLMSettings> {
    return client.put('/settings/llm', data);
  },
};
