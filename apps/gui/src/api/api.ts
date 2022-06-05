import axios, { AxiosRequestConfig } from 'axios';
import { ListStreamsResponse } from './types';

const api = () => {
  const _api = axios.create({
    baseURL: '/api',
  });
  _api.interceptors.response.use(
    (response) => response.data,
    (error) => {
      throw error.response.data;
    },
  );
  return _api;
};

export interface Invite {
  id: string;
  standardId: string;
}

const get =
  <T = any>(url: string) =>
  (): Promise<T> =>
    api().get(url);

const post =
  <T, U = any>(url: string, options?: AxiosRequestConfig<T>) =>
  (body: T): Promise<U> =>
    api().post(url, body, options);

const patch =
  <T, U = any>(url: string, options?: AxiosRequestConfig<T>) =>
  (body: T): Promise<U> =>
    api().patch(url, body, options);

export const getStreams = get<ListStreamsResponse>('/streams');
