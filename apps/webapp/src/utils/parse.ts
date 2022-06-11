export const parseEvent = async <T>(e: any) => {
  const data = e.data as Blob | ArrayBuffer;
  return JSON.parse(
    new TextDecoder().decode(
      data instanceof ArrayBuffer ? data : await (data as Blob).arrayBuffer(),
    ),
  ) as T;
};
