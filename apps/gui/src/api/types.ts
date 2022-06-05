export interface ListStreamsResponseEntry {
  streamId: string;
  viewers: number;
  uptime: number;
}
export type ListStreamsResponse = ListStreamsResponseEntry[];
