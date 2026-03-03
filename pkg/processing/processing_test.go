package processing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/censys/scan-takehome/pkg/scanning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type BadVersionData struct {
	Response string `json:response`
}

func TestProcessScan(t *testing.T) {
	assert := assert.New(t)

	// When transferred over PubSub, the data is first marshalled into JSON and then unmarshalled back to a struct
	// In order to mimic that in the test, we have to do the same here.
	v1DataString := "test response 1"
	v1ScanInitial := scanning.Scan{
		Ip:          "1.1.1.1",
		Port:        1,
		Service:     "Test Service 1",
		Timestamp:   time.Now().Unix(),
		DataVersion: scanning.V1,
	}
	v1ScanInitial.Data = &scanning.V1Data{ResponseBytesUtf8: []byte(v1DataString)}
	v1Json, err := json.Marshal(v1ScanInitial)
	require.Nil(t, err)
	var v1Scan scanning.Scan
	err = json.Unmarshal(v1Json, &v1Scan)

	v2DataString := "test response 2"
	v2ScanInitial := scanning.Scan{
		Ip:          "2.2.2.2",
		Port:        2,
		Service:     "Test Service 2",
		Timestamp:   time.Now().Unix(),
		DataVersion: scanning.V2,
	}
	v2ScanInitial.Data = &scanning.V2Data{ResponseStr: v2DataString}
	v2Json, err := json.Marshal(v2ScanInitial)
	require.Nil(t, err)
	var v2Scan scanning.Scan
	err = json.Unmarshal(v2Json, &v2Scan)

	vBadDataString := "test response 3"
	vBadScanInitial := scanning.Scan{
		Ip:          "this is a bad IP but we aren't testing for that yet",
		Port:        42,
		Service:     "Bad Service",
		Timestamp:   time.Now().Unix(),
		DataVersion: 3,
		Data:        BadVersionData{Response: vBadDataString},
	}
	vBadScanInitial.Data = &BadVersionData{Response: vBadDataString}
	vBadJson, err := json.Marshal(vBadScanInitial)
	require.Nil(t, err)
	var vBadScan scanning.Scan
	err = json.Unmarshal(vBadJson, &vBadScan)

	t.Run("Can Process v1 Scan", func(t *testing.T) {
		record, err := ProcessScan(&v1Scan)
		assert.Nil(err)
		assert.NotNil(record)

		assert.Equal(v1Scan.Ip, record.IP)
		assert.Equal(v1Scan.Port, record.Port)
		assert.Equal(v1Scan.Service, record.Service)
		assert.Equal(v1Scan.Timestamp, record.LastSeenTime.Unix())
		assert.Equal(v1DataString, record.Response)
	})
	t.Run("Can Process v2 Scan", func(t *testing.T) {
		record, err := ProcessScan(&v2Scan)
		assert.Nil(err)
		assert.NotNil(record)

		assert.Equal(v2Scan.Ip, record.IP)
		assert.Equal(v2Scan.Port, record.Port)
		assert.Equal(v2Scan.Service, record.Service)
		assert.Equal(v2Scan.Timestamp, record.LastSeenTime.Unix())
		assert.Equal(v2DataString, record.Response)
	})
	t.Run("Cannot Process bad Scan", func(t *testing.T) {
		record, err := ProcessScan(&vBadScan)
		assert.NotNil(err)
		assert.Nil(record)
	})

	t.Run("processV1 can process v1 scan", func(t *testing.T) {
		record, err := processV1Scan(&v1Scan)
		assert.Nil(err)
		assert.NotNil(record)

		assert.Equal(v1Scan.Ip, record.IP)
		assert.Equal(v1Scan.Port, record.Port)
		assert.Equal(v1Scan.Service, record.Service)
		assert.Equal(v1Scan.Timestamp, record.LastSeenTime.Unix())
		assert.Equal(v1DataString, record.Response)
	})
	t.Run("processV1 cannot process v2 scan", func(t *testing.T) {
		record, err := processV1Scan(&v2Scan)
		assert.NotNil(err)
		assert.Nil(record)
	})
	t.Run("processV1 cannot process bad scan", func(t *testing.T) {
		record, err := processV1Scan(&vBadScan)
		assert.NotNil(err)
		assert.Nil(record)
	})

	t.Run("processV2 cannot process v1 scan", func(t *testing.T) {
		record, err := processV2Scan(&v1Scan)
		assert.NotNil(err)
		assert.Nil(record)
	})
	t.Run("processV2 can process v2 scan", func(t *testing.T) {
		record, err := processV2Scan(&v2Scan)
		assert.Nil(err)
		assert.NotNil(record)

		assert.Equal(v2Scan.Ip, record.IP)
		assert.Equal(v2Scan.Port, record.Port)
		assert.Equal(v2Scan.Service, record.Service)
		assert.Equal(v2Scan.Timestamp, record.LastSeenTime.Unix())
		assert.Equal(v2DataString, record.Response)
	})
	t.Run("processV2 cannot process bad scan", func(t *testing.T) {
		record, err := processV2Scan(&vBadScan)
		assert.NotNil(err)
		assert.Nil(record)
	})
}
