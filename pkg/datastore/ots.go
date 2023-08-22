package datastore

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	conf "github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"sync"
)

var (
	otsClient    *tablestore.TableStoreClient
	once         sync.Once
	strToOtsType = map[string]tablestore.DefinedColumnType{
		"TEXT": tablestore.DefinedColumn_STRING,
		"INT":  tablestore.DefinedColumn_INTEGER,
	}
)

// example: "TEXT" to tablestore.DefinedColumn_STRING
func getOtsType(s string) tablestore.DefinedColumnType {
	return strToOtsType[s]
}

// InitOtsClient init ots client
func InitOtsClient() {
	otsClient = tablestore.NewClient(conf.ConfigGlobal.OtsEndpoint, conf.ConfigGlobal.OtsInstanceName,
		conf.ConfigGlobal.AccessKeyId, conf.ConfigGlobal.AccessKeySecret)
}

type OtsStore struct {
	config *Config
}

func NewOtsDatastore(config *Config) (*OtsStore, error) {
	// init otsClient, only first call valid
	once.Do(InitOtsClient)

	// check table is exist; if not create
	describeTableRequest := &tablestore.DescribeTableRequest{
		TableName: config.TableName,
	}
	// check table is exist or not
	if tableInfo, err := otsClient.DescribeTable(describeTableRequest); err == nil && tableInfo.TableMeta != nil {
		return &OtsStore{config: config}, nil
	}
	// create table
	createTableRequest := new(tablestore.CreateTableRequest)
	tableMeta := new(tablestore.TableMeta)
	tableMeta.TableName = config.TableName
	tableMeta.AddPrimaryKeyColumn(conf.COLPK, tablestore.PrimaryKeyType_STRING)
	for field, cate := range config.ColumnConfig {
		tableMeta.AddDefinedColumn(field, getOtsType(cate))
	}
	tableOption := new(tablestore.TableOption)
	tableOption.TimeToAlive = config.TimeToAlive
	tableOption.MaxVersion = config.MaxVersion
	reservedThroughput := new(tablestore.ReservedThroughput)
	reservedThroughput.Readcap = 0
	reservedThroughput.Writecap = 0
	createTableRequest.TableMeta = tableMeta
	createTableRequest.TableOption = tableOption
	createTableRequest.ReservedThroughput = reservedThroughput

	if _, err := otsClient.CreateTable(createTableRequest); err != nil {
		return nil, err
	}
	return &OtsStore{config: config}, nil
}

func (o *OtsStore) Get(key string, columns []string) (map[string]interface{}, error) {
	getRowRequest := new(tablestore.GetRowRequest)
	pk := new(tablestore.PrimaryKey)
	pk.AddPrimaryKeyColumn(conf.COLPK, key)
	getRowRequest.SingleRowQueryCriteria = &tablestore.SingleRowQueryCriteria{
		PrimaryKey:   pk,
		ColumnsToGet: columns,
		TableName:    o.config.TableName,
		MaxVersion:   1,
	}
	resp, err := otsClient.GetRow(getRowRequest)
	if err != nil {
		return nil, err
	}
	columnMap := resp.GetColumnMap()
	if len(columnMap.Columns) == 0 {
		return nil, nil
	}
	ret := make(map[string]interface{})
	for key, items := range columnMap.Columns {
		ret[key] = items[0].Value
	}
	return ret, nil
}

func (o *OtsStore) Put(key string, datas map[string]interface{}) error {
	putRowRequest := new(tablestore.PutRowRequest)
	putRowChange := new(tablestore.PutRowChange)
	putRowChange.TableName = o.config.TableName
	putPk := new(tablestore.PrimaryKey)
	putPk.AddPrimaryKeyColumn(conf.COLPK, key)

	putRowChange.PrimaryKey = putPk
	for col, data := range datas {
		putRowChange.AddColumn(col, data)
	}
	putRowChange.SetCondition(tablestore.RowExistenceExpectation_IGNORE)
	putRowRequest.PutRowChange = putRowChange
	if _, err := otsClient.PutRow(putRowRequest); err != nil {
		return err
	}
	return nil
}

func (o *OtsStore) Update(key string, datas map[string]interface{}) error {
	updateRowRequest := new(tablestore.UpdateRowRequest)
	updateRowChange := new(tablestore.UpdateRowChange)
	updateRowChange.TableName = o.config.TableName
	updatePk := new(tablestore.PrimaryKey)
	updatePk.AddPrimaryKeyColumn(conf.COLPK, key)
	updateRowChange.PrimaryKey = updatePk
	for col, data := range datas {
		updateRowChange.PutColumn(col, data)
	}
	updateRowChange.SetCondition(tablestore.RowExistenceExpectation_EXPECT_EXIST)
	updateRowRequest.UpdateRowChange = updateRowChange
	if _, err := otsClient.UpdateRow(updateRowRequest); err != nil {
		return err
	}
	return nil
}

func (o *OtsStore) Delete(key string) error {
	deletePk := new(tablestore.PrimaryKey)
	deletePk.AddPrimaryKeyColumn(conf.COLPK, key)
	deleteRowReq := new(tablestore.DeleteRowRequest)
	deleteRowReq.DeleteRowChange = new(tablestore.DeleteRowChange)
	deleteRowReq.DeleteRowChange.TableName = o.config.TableName
	deleteRowReq.DeleteRowChange.PrimaryKey = deletePk
	deleteRowReq.DeleteRowChange.SetCondition(tablestore.RowExistenceExpectation_EXPECT_EXIST)
	if _, err := otsClient.DeleteRow(deleteRowReq); err != nil {
		return err
	}
	return nil
}

func (o *OtsStore) ListAll(columns []string) (map[string]map[string]interface{}, error) {
	startPK := new(tablestore.PrimaryKey)
	startPK.AddPrimaryKeyColumnWithMinValue(conf.COLPK)
	endPK := new(tablestore.PrimaryKey)
	endPK.AddPrimaryKeyColumnWithMaxValue(conf.COLPK)

	rangeRowQueryCriteria := &tablestore.RangeRowQueryCriteria{
		TableName:       o.config.TableName,
		StartPrimaryKey: startPK,
		EndPrimaryKey:   endPK,
		Direction:       tablestore.FORWARD,
		MaxVersion:      1,
		Limit:           1000,
		ColumnsToGet:    columns,
	}
	getRangeRequest := &tablestore.GetRangeRequest{
		RangeRowQueryCriteria: rangeRowQueryCriteria,
	}

	getRangeResp, err := otsClient.GetRange(getRangeRequest)
	if err != nil {
		return nil, err
	}
	resp := make(map[string]map[string]interface{})
	for _, row := range getRangeResp.Rows {
		result := make(map[string]interface{})
		key := row.PrimaryKey.PrimaryKeys[0].Value.(string)
		for _, col := range row.Columns {
			result[col.ColumnName] = col.Value
		}
		resp[key] = result
	}
	return resp, nil
}

func (o *OtsStore) Close() error {
	// do nothing
	return nil
}
