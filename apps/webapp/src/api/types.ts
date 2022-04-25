export interface ListStreamsResponseEntry {
  streamId: string;
  viewers: number;
}
export type ListStreamsResponse = ListStreamsResponseEntry[];
