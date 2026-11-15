/**
 * 代码生成 API (gencode)。
 */
import { useRequest } from '~/composables/useRequest'

export interface GencodeColumnConf {
  name: string
  /** 表单控件覆盖 (空=自动推导): input/textarea/number/switch/dict/relation/date */
  kind?: string
  /** 必填覆盖 */
  required?: boolean
  inList?: boolean
  inForm?: boolean
  inSearch?: boolean
}

export interface GencodeOptions {
  table: string
  mod?: string
  title?: string
  /** 关闭组织数据权限 (默认: 表含 org_id 列即开启) */
  noScope?: boolean
  columns?: GencodeColumnConf[]
}

export function useGencodeApi() {
  const r = useRequest()
  const base = '/system/gencode'
  return {
    tables: () => r.get<{ list: Array<{ tableName: string; tableComment: string }> }>(`${base}/tables`),
    columns: (table: string) => r.get<{ list: any[] }>(`${base}/columns`, { table }),
    preview: (data: GencodeOptions) =>
      r.post<{ files: Array<{ path: string; content: string }> }>(`${base}/preview`, data),
    download: (table: string, mod: string, title: string, cols?: GencodeColumnConf[], noScope?: boolean) =>
      r.download(`${base}/download`, { table, mod, title, noScope: noScope || false, cols: cols ? JSON.stringify(cols) : '' }),
    write: (data: GencodeOptions) =>
      r.post<{ written: string[] }>(`${base}/write`, data),
  }
}
