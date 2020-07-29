package migration

import (
	"database/sql"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/pressly/goose"
)

func initialize20200727144157() {
	if _, ok := initializedMigrations[20200727144157]; !ok {
		goose.AddMigration(Up20200727144157, Down20200727144157)
		initializedMigrations[20200727144157] = true
	}
}

func Up20200727144157(tx *sql.Tx) error {
	// This code is executed when the migration is applied.

	client, err := getClient()
	if err != nil {
		return err
	}

	migrationsRan, err := getRanSQLMigrations(client)
	if err != nil {
		return err
	}

	if _, ok := migrationsRan[20200727144157]; ok {
		return nil
	}

	pageSize := int32(100)
	page := int32(0)
	paginator := pagination.NewRequest(page, pageSize)
	wtvsResults := -1
	for wtvsResults != 0 {
		wtvs, err := client.ListWorkflowTemplateVersionsDBAll(&paginator)
		if err != nil {
			return err
		}
		//Exit condition; Check for more results
		wtvsResults = len(wtvs)
		if wtvsResults > 0 {
			page++
			paginator = pagination.NewRequest(page, pageSize)
		}

		for _, wtv := range wtvs {
			params, err := v1.ParseParametersFromManifest([]byte(wtv.Manifest))
			if err != nil {
				return err
			}
			wtv.Parameters = params
			err = client.UpdateWorkflowTemplateVersionDB(wtv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Down20200727144157(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
