package service

// ProtocolModelingPagination 描述协议建模变量列表的服务端分页状态。
type ProtocolModelingPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

func newProtocolModelingPagination(page, pageSize, total int) ProtocolModelingPagination {
	return ProtocolModelingPagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
	}
}
