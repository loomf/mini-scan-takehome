package processing

import "time"

// IPRecord is the processed record of a scan result, used to store the scan result in the database
type IPRecord struct {
	ID           uint      `gorm:"column:id;primaryKey;autoIncrement"`
	IP           string    `gorm:"column:ip;size:255;uniqueIndex:ip_port_service"`
	Port         uint32    `gorm:"column:port;uniqueIndex:ip_port_service"`
	Service      string    `gorm:"column:service;size:255;uniqueIndex:ip_port_service"`
	Response     string    `gorm:"column:response;type:text"`
	LastSeenTime time.Time `gorm:"column:last_seen_time"`
}
