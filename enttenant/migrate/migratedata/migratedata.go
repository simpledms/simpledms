package migratedata

import (
	"log"

	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect/sql/schema"

	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/attribute"
	"github.com/simpledms/simpledms/model/common/attributetype"
)

/*
import (
	"context"
	"log"

	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"

	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/attribute"
	"github.com/simpledms/simpledms/model/common/attributetype"
)

func RenamePropertyToField(dir *sqltool.GolangMigrateDir) error {
	w := &schema.DirWriter{Dir: dir}
	client := enttenant.NewClient(enttenant.Driver(schema.NewWriteDriver(dialect.SQLite, w)))

	err := client.Attribute.Update().
		SetType(attributetype.Field).
		Where(
			attribute.TypeEQ(attributetype.Property),
		).
		Exec(context.Background())
	if err != nil {
		log.Println(err)
		return err
	}

	return w.FlushChange("rename_property_to_field",
		"Rename Property to Field")
}
*/
