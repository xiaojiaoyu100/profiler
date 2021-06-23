package profile

import (
	"testing"

	"github.com/aliyun/aliyun-tablestore-go-sdk/v5/tablestore"
)

func TestUploadPath(t *testing.T) {
	t.Logf(UploadPath("abc", "bcf", "cpu", "efg"))
}

func TestUnMarshalProfileRow(t *testing.T) {
	row := new(tablestore.Row)
	row.Columns = append(row.Columns, &tablestore.AttributeColumn{
		ColumnName: "profile_id",
		Value:      "dfdfdkfmdkfkdfkdm",
	}, &tablestore.AttributeColumn{
		ColumnName: "size",
		Value:      int64(64),
	})
	unMarshalProfileRow(row)
}
