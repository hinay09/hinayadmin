// Package excelx Excel 通用导入导出工具 (基于 excelize)。
//
// 导出: Build 构建单 sheet 工作簿 + WriteToResponse 附件下载 (绕过统一 JSON 响应包装,
// GoFrame MiddlewareHandlerResponse 检测到响应缓冲已有内容时不再包装)。
// 导入: Read 将上传的 xlsx 解析为字符串二维表, 表头校验与行解析由调用方完成。
package excelx

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

// ContentType xlsx 的标准 MIME 类型。
const ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Build 构建单 sheet 的 xlsx 文件。
// sheet 为工作表名, headers 为表头行, rows 为数据行; 表头加粗, 列宽按内容自适应。
func Build(sheet string, headers []string, rows [][]any) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// 默认工作表名重命名为业务 sheet 名
	if sheet != "" && sheet != "Sheet1" {
		if err := f.SetSheetName("Sheet1", sheet); err != nil {
			return nil, err
		}
	}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return nil, err
		}
	}
	if styleID, serr := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}}); serr == nil {
		_ = f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), styleID)
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return nil, err
			}
		}
	}
	autoWidth(f, sheet, headers, rows)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Read 解析 xlsx 字节流, 返回第一个 sheet 的全部行 (含表头, 单元格统一为字符串)。
func Read(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.GetRows(f.GetSheetName(0))
}

// WriteToResponse 将 xlsx 内容作为附件写入 HTTP 响应。
// 文件名支持中文 (RFC 5987 filename* + ASCII 回退)。
func WriteToResponse(ctx context.Context, filename string, content []byte) {
	r := g.RequestFromCtx(ctx)
	if r == nil {
		return
	}
	r.Response.Header().Set("Content-Type", ContentType)
	r.Response.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s",
			url.PathEscape(filename), url.PathEscape(filename)))
	r.Response.Write(content)
}

// cellName 由列数与行号生成单元格名。
func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// autoWidth 按表头与单元格内容估算列宽 (中文字符按 2 计), 上限 50。
func autoWidth(f *excelize.File, sheet string, headers []string, rows [][]any) {
	widths := make([]float64, len(headers))
	runes := func(s string) float64 {
		w := 0.0
		for _, r := range s {
			if r > 0x7F {
				w += 2
			} else {
				w++
			}
		}
		return w
	}
	for i, h := range headers {
		widths[i] = runes(h) + 4
	}
	for _, row := range rows {
		for i := 0; i < len(row) && i < len(widths); i++ {
			if w := runes(fmt.Sprintf("%v", row[i])) + 2; w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i, w := range widths {
		if w > 50 {
			w = 50
		}
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, w)
	}
}
