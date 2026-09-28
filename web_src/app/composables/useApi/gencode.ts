/**
 * 代码生成 API (gencode)。
 */
import { useRequest } from '~/composables/useRequest'

export interface GencodeColumnConf {
  name: string
  inList?: boolean
  inForm?: boolean
  inSearch?: boolean
}

export function useGencodeApi() {
  const r = useRequest()
  const base = '/system/gencode'
  return {
    tables: () => r.get<{ list: Array<{ tableName: string; tableComment: string }> }>(`${base}/tables`),
    columns: (table: string) => r.get<{ list: any[] }>(`${base}/columns`, { table }),
    preview: (data: { table: string; mod?: string; title?: string; columns?: GencodeColumnConf[] }) =>
      r.post<{ files: Array<{ path: string; content: string }> }>(`${base}/preview`, data),
    download: (table: string, mod: string, title: string, cols?: GencodeColumnConf[]) =>
      r.download(`${base}/download`, { table, mod, title, cols: cols ? JSON.stringify(cols) : '' }),
    write: (data: { table: string; mod?: string; title?: string; columns?: GencodeColumnConf[] }) =>
      r.post<{ written: string[] }>(`${base}/write`, data),
  }
}
