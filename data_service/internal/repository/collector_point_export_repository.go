package repository

import (
	"context"
	"fmt"
	"strings"
)

type CollectorPointExportFilter struct {
	Scope           string
	GroupID         *string
	IncludeChildren bool
	Search          string
	DataType        string
	Enabled         *bool
	SortBy          string
	SortOrder       string
	Page            int
	PageSize        int
	Pages           []int64
	PointIDs        []string
}

type CollectorPointExportRecord struct {
	CollectorPointRecord
	GroupPath string
}

func (r *CollectorRepository) StreamPointsForExport(ctx context.Context, projectID, connectionID string, filter CollectorPointExportFilter, visit func(CollectorPointExportRecord) error) error {
	args := []any{projectID, connectionID}
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	conditions := []string{"p.project_id=$1", "p.connection_id=$2"}
	groupCTE := ""
	if filter.Scope == "group" {
		if filter.GroupID == nil {
			// 未选择具体分组时表示导出当前连接的全部变量。
		} else if filter.IncludeChildren {
			groupPlaceholder := add(*filter.GroupID)
			groupCTE = fmt.Sprintf(`, selected_groups AS (
SELECT id FROM data_collector_point_groups WHERE project_id=$1 AND connection_id=$2 AND id=%s
UNION ALL
SELECT child.id FROM data_collector_point_groups child JOIN selected_groups parent ON child.parent_id=parent.id WHERE child.project_id=$1 AND child.connection_id=$2
)`, groupPlaceholder)
			conditions = append(conditions, "p.group_id IN (SELECT id FROM selected_groups)")
		} else {
			conditions = append(conditions, "p.group_id="+add(*filter.GroupID))
		}
	} else if filter.GroupID != nil {
		conditions = append(conditions, "p.group_id="+add(*filter.GroupID))
	}
	if filter.Scope == "selected" {
		conditions = append(conditions, "p.id=ANY("+add(filter.PointIDs)+"::uuid[])")
	}
	if value := strings.TrimSpace(filter.Search); value != "" && filter.Scope != "group" && filter.Scope != "selected" {
		conditions = append(conditions, "(p.name ILIKE "+add("%"+value+"%")+" OR p.code ILIKE "+fmt.Sprintf("$%d", len(args))+" OR p.address_text ILIKE "+fmt.Sprintf("$%d", len(args))+")")
	}
	if value := strings.TrimSpace(filter.DataType); value != "" && filter.Scope != "group" && filter.Scope != "selected" {
		conditions = append(conditions, "p.data_type="+add(value))
	}
	if filter.Enabled != nil && filter.Scope != "group" && filter.Scope != "selected" {
		conditions = append(conditions, "p.enabled="+add(*filter.Enabled))
	}
	sortColumn := map[string]string{"name": "name", "dataType": "data_type", "enabled": "enabled", "createdAt": "created_at", "sortOrder": "sort_order"}[filter.SortBy]
	if sortColumn == "" {
		sortColumn = "sort_order"
	}
	direction := "ASC"
	if strings.EqualFold(filter.SortOrder, "desc") {
		direction = "DESC"
	}
	pageCondition := "TRUE"
	if filter.Scope == "current_page" {
		start := (filter.Page-1)*filter.PageSize + 1
		pageCondition = fmt.Sprintf("row_num BETWEEN %s AND %s", add(start), add(start+filter.PageSize-1))
	} else if filter.Scope == "pages" {
		pageSizePlaceholder := add(filter.PageSize)
		pagesPlaceholder := add(filter.Pages)
		pageCondition = fmt.Sprintf("(((row_num-1)/%s)+1)=ANY(%s::bigint[])", pageSizePlaceholder, pagesPlaceholder)
	}
	query := fmt.Sprintf(`WITH RECURSIVE group_tree AS (
SELECT id,parent_id,name,name::text AS path FROM data_collector_point_groups WHERE project_id=$1 AND connection_id=$2 AND parent_id IS NULL
UNION ALL
SELECT child.id,child.parent_id,child.name,(parent.path || '/' || child.name) FROM data_collector_point_groups child JOIN group_tree parent ON child.parent_id=parent.id WHERE child.project_id=$1 AND child.connection_id=$2
)%s, filtered AS (
SELECT p.id,p.project_id,p.connection_id,p.group_id,p.code,p.name,p.description,p.address,p.address_text,p.address_schema_version,p.data_type,p.element_count,p.read_options,p.acquisition,p.enabled,p.sort_order,p.metadata,p.created_at,p.updated_at,COALESCE(group_tree.path,'') AS group_path
FROM data_collector_points p LEFT JOIN group_tree ON group_tree.id=p.group_id
WHERE %s
), ranked AS (
SELECT filtered.*,row_number() OVER (ORDER BY %s %s,id ASC) AS row_num FROM filtered
)
SELECT id,project_id,connection_id,group_id,code,name,description,address,address_text,address_schema_version,data_type,element_count,read_options,acquisition,enabled,sort_order,metadata,created_at,updated_at,group_path
FROM ranked WHERE %s ORDER BY row_num`, groupCTE, strings.Join(conditions, " AND "), sortColumn, direction, pageCondition)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return wrapUnifiedCollectorRepositoryError("查询导出采集变量失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var record CollectorPointExportRecord
		if err := rows.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Code, &record.Name, &record.Description, &record.Address, &record.AddressText, &record.AddressSchemaVersion, &record.DataType, &record.ElementCount, &record.ReadOptions, &record.Acquisition, &record.Enabled, &record.SortOrder, &record.Metadata, &record.CreatedAt, &record.UpdatedAt, &record.GroupPath); err != nil {
			return err
		}
		if err := visit(record); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return wrapUnifiedCollectorRepositoryError("遍历导出采集变量失败", err)
	}
	return nil
}
