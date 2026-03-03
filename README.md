# Mini-Scan

Hello!

As you've heard by now, Censys scans the internet at an incredible scale. Processing the results necessitates scaling horizontally across thousands of machines. One key aspect of our architecture is the use of distributed queues to pass data between machines.

---

The `docker-compose.yml` file sets up a toy example of a scanner. It spins up a Google Pub/Sub emulator, creates a topic and subscription, and publishes scan results to the topic. It can be run via `docker compose up`.

Your job is to build the data processing side. It should:

1. Pull scan results from the subscription `scan-sub`.
2. Maintain an up-to-date record of each unique `(ip, port, service)`. This should contain when the service was last scanned and a string containing the service's response.

> **_NOTE_**
> The scanner can publish data in two formats, shown below. In both of the following examples, the service response should be stored as: `"hello world"`.
>
> ```javascript
> {
>   // ...
>   "data_version": 1,
>   "data": {
>     "response_bytes_utf8": "aGVsbG8gd29ybGQ="
>   }
> }
>
> {
>   // ...
>   "data_version": 2,
>   "data": {
>     "response_str": "hello world"
>   }
> }
> ```

Your processing application should be able to be scaled horizontally, but this isn't something you need to actually do. The processing application should use `at-least-once` semantics where ever applicable.

You may write this in any languages you choose, but Go would be preferred.

You may use any data store of your choosing, with `sqlite` being one example. Like our own code, we expect the code structure to make it easy to switch data stores.

Please note that Google Pub/Sub is best effort ordering and we want to keep the latest scan. While the example scanner does not publish scans at a rate where this would be an issue, we expect the application to be able to handle extreme out of orderness. Consider what would happen if the application received a scan that is 24 hours old.

cmd/scanner/main.go should not be modified

---

Please upload the code to a publicly accessible GitHub, GitLab or other public code repository account. This README file should be updated, briefly documenting your solution. Like our own code, we expect testing instructions: whether it’s an automated test framework, or simple manual steps.

To help set expectations, we believe you should aim to take no more than 4 hours on this task.

We understand that you have other responsibilities, so if you think you’ll need more than 5 business days, just let us know when you expect to send a reply.

Please don’t hesitate to ask any follow-up questions for clarification.

---

## Solution

The solution is split into two parts, the processor command which handles connectivity to the database and subscription, and the processing package which contains the database model as well as the logic for processing the different scan versions. These are separated to support automated testing for the processing logic without having to interact with external services. 

### Processing Package
The processing logic is contained in the pkg/processing package. It contains 3 files:
* model.go contains the database model for a record
* processing.go handles the processing logic itself, there is one public function called ProcessScan which takes a scanning.Scan object and returns a processing.IPRecord object and a possible error. This function acts as a simple wrapper around the two private functions which handle the processing of the v1 and v2 scan records respectively. The processing logic for the different versions is separated into their own functions in order to allow easy extensibility and testing, and the functions themselves are private because the end user shouldn't need to concern themselves with the version so long as the data is valid.
The tests cover the most common cases: Good data of either data type being processed correctly via the public function, as well as bad data returning an error. The tests also cover the private functions for handling each data type specifically, to ensure that we error appropriately in the event that the wrong processing function is somehow called.
The tests can be called by running `go test ./pkg/processing/` from this directory.

### Processor Command
To run the processor, you can use the following command: `PUBSUB_EMULATOR_HOST=localhost:8085 go run cmd/processor/main.go` from this directory.
The processor command contains the logic for comminucating with outside services. In this case, PubSub and a SQLite database. The core of the logic exists in the subscription's reciever function, which attempts to unmarshal the pubsub message into a scaning.Scan object, process that into an IPRecord object and then insert that IPRecord into the database. The database table has a unique index on the ip-port-service triad to prevent repeated entries. In order to ensure that only the most up to date information is stored, we first check for the existence of an entry, and then either create or update as necessary. This is done within a transaction to prevent any other processes from making changes while we are in the process of determining whether our data needs to be stored. This should allow for deployment of additional versions of the command without causing conflicts.

### Concurrency
Since we are using `subscription.Recieve` we are already working in a multithreaded capacity. However, there are a number of considerations to be made should we want to scale horizontally with additional workers in addition. The two main concerns for horizontal scaling are database locking and message acknowledgement.

For database locking, SQLite is a poor choice if we are going to continue to scale horizontally since it will lock the entire database for writes. If I were to develop this for a production environment, I would recommend using a different database like MySQL or PostreSQL which allow for row-level locking. Beyond that, the only change that would be needed to the code would be to remove the `	genericDB.SetMaxOpenConns(1)` call on line 41 of cmd/processor/main.go

The other issue for concurrency is to avoid unnecessary reprocessing of messages. If multiple processes are reading from the same PubSub Subscription, each process will recieve un-acknowledged messages from the queue. In the code currently, a message is only acknowledged _after_ it has been successfully inserted into the database. This ensures that data will either make it into the database, or be sent for re-processing. If we want to optimize for throughput and are comfortable with the risk of potentially losing data if there are database connectivity issues, we could acknowledge the message as soon as it is recieved by the processor. This would reduce the number of messages that are processed repeatedly. However, the processor is already capable of handling reprocessed messages without issue, and given the risk of data loss, I would recommend against that unless improved performance is badly needed.