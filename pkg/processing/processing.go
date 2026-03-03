package processing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/censys/scan-takehome/pkg/scanning"
)

func ProcessScan(scan *scanning.Scan) (*IPRecord, error) {

	// Separate the data processing logic into separate functions for each data version for easy extensibility
	switch scan.DataVersion {
	case scanning.V1:
		ipRecord, err := processV1Scan(scan)
		if err != nil {
			return nil, err
		}
		return ipRecord, nil
	case scanning.V2:
		ipRecord, err := processV2Scan(scan)
		if err != nil {
			return nil, err
		}
		return ipRecord, nil
	}
	return nil, fmt.Errorf("invalid data version: %d", scan.DataVersion)
}

func processV1Scan(scan *scanning.Scan) (*IPRecord, error) {
	if scan.DataVersion != scanning.V1 {
		return nil, fmt.Errorf("invalid data version: %d, expected: %d", scan.DataVersion, scanning.V1)
	}

	// When we unmarshal the JSON for the overall Scan, we can't directly cast the data
	// Instead, we convert the Data field back to JSON and then cast it to the appropriate data version
	jsonStr, err := json.Marshal(scan.Data.(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to cast scan.Data to json, invalid data: %v", scan.Data)
	}
	var v1Data scanning.V1Data
	err = json.Unmarshal(jsonStr, &v1Data)
	if err != nil {
		return nil, fmt.Errorf("failed to cast scan.Data to to scanning.V1Data, invalid data: %v", scan.Data)
	}

	ipRecord := &IPRecord{
		IP:           scan.Ip,
		Port:         scan.Port,
		Service:      scan.Service,
		Response:     string(v1Data.ResponseBytesUtf8),
		LastSeenTime: time.Unix(scan.Timestamp, 0),
	}
	return ipRecord, nil
}

func processV2Scan(scan *scanning.Scan) (*IPRecord, error) {
	if scan.DataVersion != scanning.V2 {
		return nil, fmt.Errorf("invalid data version: %d, expected: %d", scan.DataVersion, scanning.V2)
	}

	// When we unmarshal the JSON for the overall Scan, we can't directly cast the data
	// Instead, we convert the Data field back to JSON and then cast it to the appropriate data version
	jsonStr, err := json.Marshal(scan.Data.(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to cast scan.Data to json, invalid data: %v", scan.Data)
	}
	var v2Data scanning.V2Data
	err = json.Unmarshal(jsonStr, &v2Data)
	if err != nil {
		return nil, fmt.Errorf("failed to cast scan.Data to to scanning.V1Data, invalid data: %v", scan.Data)
	}
	ipRecord := &IPRecord{
		IP:           scan.Ip,
		Port:         scan.Port,
		Service:      scan.Service,
		Response:     v2Data.ResponseStr,
		LastSeenTime: time.Unix(scan.Timestamp, 0),
	}
	return ipRecord, nil
}
