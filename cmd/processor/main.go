package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/censys/scan-takehome/pkg/processing"
	"github.com/censys/scan-takehome/pkg/scanning"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {

	projectId := flag.String("project", "test-project", "GCP Project ID")
	subscriptionId := flag.String("subscription", "scan-sub", "GCP PubSub Subscription ID")
	dbPath := flag.String("db-path", "test.db", "Path to the database file")

	ctx := context.Background()

	fmt.Printf("ProjectID: %s\nSubscription: %s\nDB Path: %s\n", *projectId, *subscriptionId, *dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Database setup
	// We are using GORM to simplify the database operations as well as allow for easy switching of the database engine
	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("failed to open sqlite database: %w", err))
	}
	genericDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("failed to set get raw sqlite DB handler: %w", err))
	}
	// This is necessary to prevent locking while using SQLite
	genericDB.SetMaxOpenConns(1)

	// Ensure that the IP Record table exists
	err = db.AutoMigrate(&processing.IPRecord{})
	if err != nil {
		panic(fmt.Errorf("failed to create IP Record table: %w", err))
	}

	// PubSub setup
	client, err := pubsub.NewClient(ctx, *projectId)
	if err != nil {
		panic(fmt.Errorf("failed to create pubsub client: %w", err))
	}

	sub := client.Subscription(*subscriptionId)

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var scanEntry scanning.Scan
		err := json.Unmarshal(msg.Data, &scanEntry)
		if err != nil {
			// If we can't unmarshal the json, log an error and continue
			logger.Error(fmt.Sprintf("failed to unmarshal scan: %v", err))
			// Since this is an unrecoverable error, we should acknowledge the message to avoid wasted effort reprocessing
			msg.Ack()
			return
		}

		ipRecord, err := processing.ProcessScan(&scanEntry)
		if err != nil {
			// If we can't process the scan, log an error and continue
			logger.Error(fmt.Sprintf("failed to process scan: %v", err))
			// Since this is an unrecoverable error, we should acknowledge the message to avoid wasted effort reprocessing
			msg.Ack()
			return
		}

		maxRetries := 5
		transactionSucceeded := false
		for retries := 0; retries < maxRetries; retries++ {
			err = db.Transaction(func(tx *gorm.DB) error {
				records := []processing.IPRecord{}
				err := tx.
					Where("ip = ?", ipRecord.IP).
					Where("port = ?", ipRecord.Port).
					Where("service = ?", ipRecord.Service).
					Find(&records).
					Error
				if err != nil {
					return err
				}

				if len(records) > 0 {
					if len(records) > 1 {
						// Our database table should have a unique index making this impossible. Log an error, but continue.
						logger.Error(fmt.Sprintf("Multiple database entries found for IP %s, Port %d, Service %s", ipRecord.IP, ipRecord.Port, ipRecord.Service))
					}

					// We already have an entry for this ip-port-service, we should update it if the new last_seen_time > the old one
					oldRecord := records[0]
					if oldRecord.LastSeenTime.Before(ipRecord.LastSeenTime) {
						// Our new record is more recent, update the last_seen_time and response
						return tx.
							Model(&oldRecord).
							Updates(map[string]interface{}{
								"last_seen_time": ipRecord.LastSeenTime,
								"response":       ipRecord.Response,
							}).
							Error
					}
				} else {
					// We don't have a record yet, create one.
					return tx.Create(ipRecord).Error
				}
				return nil
			})

			if err == nil {
				transactionSucceeded = true
				break
			}
		}
		if transactionSucceeded {
			fmt.Printf("Successfully stored record: %+v\n", *ipRecord)
			msg.Ack()
		} else {
			logger.Error(fmt.Sprintf("failed to insert scan: %v", err))
			// Since this is a database issue and not an issue with the message, unacknowledge it so that it can be retried later
			msg.Nack()
		}
	})
}
