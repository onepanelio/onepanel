package v1

import (
	sq "github.com/Masterminds/squirrel"
)

func (c *Client) InsertLabelsBuilder(resource string, resourceId uint64, keyValues map[string]string) sq.InsertBuilder {
	sb := sb.Insert("labels").
		Columns("resource", "resource_id", "key", "value")

	for key, value := range keyValues {
		sb = sb.Values(resource, resourceId, key, value)
	}

	return sb
}

func (c *Client) GetDbLabels(resource string, ids ...uint64) (labels []*Label, err error) {
	if len(ids) == 0 {
		return make([]*Label, 0), nil
	}

	tx, err := c.DB.Begin()
	if err != nil {
		return nil, err
	}

	whereIn := "resource_id IN (?"
	for i := range ids {
		if i == 0 {
			continue
		}

		whereIn += ",?"
	}
	whereIn += ")"

	defer tx.Rollback()

	query, args, err := sb.Select("id", "key", "value", "resource", "resource_id").
		From("labels").
		Where(whereIn, ids).
		Where(sq.Eq{
			"resource": resource,
		}).
		OrderBy("key").
		ToSql()

	if err != nil {
		return nil, err
	}

	allArgs := make([]interface{}, 0)
	for _, arg := range args[0].([]uint64) {
		allArgs = append(allArgs, arg)
	}
	allArgs = append(allArgs, args[1])

	err = c.DB.Select(&labels, query, allArgs...)
	if err != nil {
		return nil, err
	}

	return
}

func (c *Client) GetDbLabelsMapped(resource string, ids ...uint64) (result map[uint64]map[string]string, err error) {
	dbLabels, err := c.GetDbLabels(resource, ids...)
	if err != nil {
		return
	}

	result = make(map[uint64]map[string]string)
	for _, dbLabel := range dbLabels {
		_, ok := result[dbLabel.ResourceId]
		if !ok {
			result[dbLabel.ResourceId] = make(map[string]string)
		}
		result[dbLabel.ResourceId][dbLabel.Key] = dbLabel.Value
	}

	return
}
