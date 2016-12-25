package s

type S struct {
	F1 string            `table_name:"t1" column_name:"c1"`
	F2 string            `table_name:"t1" column_name:"c2"`
	F3 int               `table_name:"t1" column_name:"c3"`
	F4 map[string]string `table_name:"site_section_group_attribute_data" column_name:""`
	F5 struct {
		Items []struct {
			ID int64 `table_name:"site_section_group_relation" column_name:"site_section_group_parent_id"`
		}
	}
}
